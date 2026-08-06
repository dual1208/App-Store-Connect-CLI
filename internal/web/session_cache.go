package web

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dual1208/App-Store-Connect-CLI/internal/privatefile"
)

const (
	webSessionCacheEnabledEnv = "ASC_WEB_SESSION_CACHE"
	webSessionCacheDirEnv     = "ASC_WEB_SESSION_CACHE_DIR"
	webSessionCacheVersion    = 1
)

var ErrCachedSessionExpired = errors.New("cached web session expired")

type persistedSession struct {
	Version   int                  `json:"version"`
	UpdatedAt time.Time            `json:"updated_at"`
	UserEmail string               `json:"user_email,omitempty"`
	Cookies   map[string][]pCookie `json:"cookies"`
}

type pCookie struct {
	Name     string    `json:"name"`
	Value    string    `json:"value"`
	Path     string    `json:"path,omitempty"`
	Domain   string    `json:"domain,omitempty"`
	Expires  time.Time `json:"expires,omitempty"`
	MaxAge   int       `json:"max_age,omitempty"`
	Secure   bool      `json:"secure,omitempty"`
	HttpOnly bool      `json:"http_only,omitempty"`
	SameSite int       `json:"same_site,omitempty"`
}

type persistedLastSession struct {
	Version int    `json:"version"`
	Key     string `json:"key"`
}

var sessionInfoFetcher = getSessionInfo

func webSessionCacheEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(webSessionCacheEnabledEnv))) {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

func webSessionCacheDir() (string, error) {
	if custom := strings.TrimSpace(os.Getenv(webSessionCacheDirEnv)); custom != "" {
		return custom, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	return filepath.Join(home, ".asc", "web-sessions"), nil
}

func webSessionCacheKey(username string) string {
	normalized := strings.ToLower(strings.TrimSpace(username))
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

func webSessionFilePath(key string) (string, error) {
	dir, err := webSessionCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "session-"+key+".json"), nil
}

func webSessionLastFilePath() (string, error) {
	dir, err := webSessionCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "last.json"), nil
}

func sessionCookieURLs() []*url.URL {
	return []*url.URL{
		{Scheme: "https", Host: "appstoreconnect.apple.com", Path: "/"},
		{Scheme: "https", Host: "idmsa.apple.com", Path: "/"},
		{Scheme: "https", Host: "gsa.apple.com", Path: "/"},
	}
}

func isExpiredCookie(c pCookie, now time.Time) bool {
	return c.MaxAge < 0 || (!c.Expires.IsZero() && c.Expires.Before(now))
}

func serializeCookieJar(jar http.CookieJar, userEmail string) persistedSession {
	now := time.Now().UTC()
	out := persistedSession{Version: webSessionCacheVersion, UpdatedAt: now, UserEmail: strings.TrimSpace(userEmail), Cookies: map[string][]pCookie{}}
	for _, base := range sessionCookieURLs() {
		for _, cookie := range jar.Cookies(base) {
			if cookie == nil || cookie.Name == "" {
				continue
			}
			entry := pCookie{Name: cookie.Name, Value: cookie.Value, Path: cookie.Path, Domain: cookie.Domain, Expires: cookie.Expires, MaxAge: cookie.MaxAge, Secure: cookie.Secure, HttpOnly: cookie.HttpOnly, SameSite: int(cookie.SameSite)}
			if !isExpiredCookie(entry, now) {
				out.Cookies[base.String()] = append(out.Cookies[base.String()], entry)
			}
		}
	}
	return out
}

func hydrateCookieJar(jar http.CookieJar, sess persistedSession) int {
	now := time.Now().UTC()
	loaded := 0
	for rawURL, entries := range sess.Cookies {
		base, err := url.Parse(rawURL)
		if err != nil || base.Scheme != "https" {
			continue
		}
		cookies := make([]*http.Cookie, 0, len(entries))
		for _, entry := range entries {
			if entry.Name == "" || isExpiredCookie(entry, now) {
				continue
			}
			cookies = append(cookies, &http.Cookie{Name: entry.Name, Value: entry.Value, Path: entry.Path, Domain: entry.Domain, Expires: entry.Expires, MaxAge: entry.MaxAge, Secure: entry.Secure, HttpOnly: entry.HttpOnly, SameSite: http.SameSite(entry.SameSite)})
		}
		jar.SetCookies(base, cookies)
		loaded += len(cookies)
	}
	return loaded
}

func writeJSONFile(path string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return privatefile.WriteAtomically(path, data)
}

func readJSONFile(path string, value any) (bool, error) {
	dir := filepath.Dir(path)
	if err := privatefile.EnsureDir(dir); err != nil {
		return false, err
	}
	data, err := privatefile.Read(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(data, value); err != nil {
		return false, fmt.Errorf("parse private session file: %w", err)
	}
	return true, nil
}

func writeSessionToFile(key string, sess persistedSession) error {
	path, err := webSessionFilePath(key)
	if err != nil {
		return err
	}
	if err := writeJSONFile(path, sess); err != nil {
		return err
	}
	lastPath, err := webSessionLastFilePath()
	if err != nil {
		return err
	}
	return writeJSONFile(lastPath, persistedLastSession{Version: webSessionCacheVersion, Key: key})
}

func readSessionFromFile(key string) (persistedSession, bool, error) {
	path, err := webSessionFilePath(key)
	if err != nil {
		return persistedSession{}, false, err
	}
	var sess persistedSession
	ok, err := readJSONFile(path, &sess)
	if err != nil || !ok {
		return persistedSession{}, ok, err
	}
	if sess.Version != webSessionCacheVersion {
		return persistedSession{}, false, fmt.Errorf("unsupported web session cache version %d", sess.Version)
	}
	return sess, true, nil
}

func readLastSessionFromFile() (persistedSession, bool, error) {
	path, err := webSessionLastFilePath()
	if err != nil {
		return persistedSession{}, false, err
	}
	var last persistedLastSession
	ok, err := readJSONFile(path, &last)
	if err != nil || !ok || last.Key == "" {
		return persistedSession{}, false, err
	}
	return readSessionFromFile(last.Key)
}

func resumeFromPersistedSession(ctx context.Context, sess persistedSession) (*AuthSession, bool, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, false, err
	}
	if hydrateCookieJar(jar, sess) == 0 {
		return nil, false, nil
	}
	client := newWebHTTPClient(jar)
	info, err := sessionInfoFetcher(ctx, client)
	if err != nil {
		if isSessionInfoAuthExpired(err) {
			return nil, false, ErrCachedSessionExpired
		}
		return nil, false, nil
	}
	session := &AuthSession{Client: client}
	applySessionInfo(session, info)
	return session, true, nil
}

func loadSessionFromPersistedSession(sess persistedSession) (*AuthSession, bool, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, false, err
	}
	if hydrateCookieJar(jar, sess) == 0 {
		return nil, false, nil
	}
	return &AuthSession{Client: newWebHTTPClient(jar), UserEmail: strings.TrimSpace(sess.UserEmail)}, true, nil
}

func PersistSession(session *AuthSession) error {
	if !webSessionCacheEnabled() || session == nil || session.Client == nil || session.Client.Jar == nil || strings.TrimSpace(session.UserEmail) == "" {
		return nil
	}
	key := webSessionCacheKey(session.UserEmail)
	return writeSessionToFile(key, serializeCookieJar(session.Client.Jar, session.UserEmail))
}

func LoadCachedSession(username string) (*AuthSession, bool, error) {
	if !webSessionCacheEnabled() || strings.TrimSpace(username) == "" {
		return nil, false, nil
	}
	sess, ok, err := readSessionFromFile(webSessionCacheKey(username))
	if err != nil || !ok {
		return nil, false, err
	}
	return loadSessionFromPersistedSession(sess)
}

func TryResumeSession(ctx context.Context, username string) (*AuthSession, bool, error) {
	if !webSessionCacheEnabled() || strings.TrimSpace(username) == "" {
		return nil, false, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	sess, ok, err := readSessionFromFile(webSessionCacheKey(username))
	if err != nil || !ok {
		return nil, false, err
	}
	resumed, ok, err := resumeFromPersistedSession(ctx, sess)
	if err == nil && ok && resumed != nil {
		_ = PersistSession(resumed)
	}
	return resumed, ok, err
}

func LoadLastCachedSession() (*AuthSession, bool, error) {
	if !webSessionCacheEnabled() {
		return nil, false, nil
	}
	sess, ok, err := readLastSessionFromFile()
	if err != nil || !ok {
		return nil, false, err
	}
	return loadSessionFromPersistedSession(sess)
}

func TryResumeLastSession(ctx context.Context) (*AuthSession, bool, error) {
	if !webSessionCacheEnabled() {
		return nil, false, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	sess, ok, err := readLastSessionFromFile()
	if err != nil || !ok {
		return nil, false, err
	}
	resumed, ok, err := resumeFromPersistedSession(ctx, sess)
	if err == nil && ok && resumed != nil {
		_ = PersistSession(resumed)
	}
	return resumed, ok, err
}

func DeleteSession(username string) error {
	if strings.TrimSpace(username) == "" {
		return nil
	}
	key := webSessionCacheKey(username)
	path, err := webSessionFilePath(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	lastPath, err := webSessionLastFilePath()
	if err != nil {
		return err
	}
	var last persistedLastSession
	ok, readErr := readJSONFile(lastPath, &last)
	if readErr == nil && ok && last.Key == key {
		if err := os.Remove(lastPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func DeleteAllSessions() error {
	dir, err := webSessionCacheDir()
	if err != nil {
		return err
	}
	if _, err := os.Lstat(dir); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err := privatefile.EnsureDir(dir); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Type().IsRegular() && (entry.Name() == "last.json" || strings.HasPrefix(entry.Name(), "session-")) {
			if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}
