package cmd

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/dual1208/App-Store-Connect-CLI/internal/cli/shared"
	"github.com/dual1208/App-Store-Connect-CLI/internal/cli/shared/suggest"
)

type invocationShape uint8

const (
	invocationShapeLeaf invocationShape = iota
	invocationShapeBareGroup
	invocationShapeGroupWithFlags
	invocationShapeUnknownChild
)

type invocationAnalysis struct {
	command      *ffcli.Command
	shape        invocationShape
	unknownToken string
	unknownFlag  bool
}

func analyzeInvocation(root *ffcli.Command, args []string) invocationAnalysis {
	current := root
	sawFlag := false

	for i := 0; i < len(args); {
		token := args[i]
		if token == "" {
			i++
			continue
		}
		if sub := findDirectSubcommand(current, token); sub != nil {
			current = sub
			i++
			continue
		}
		if isHelpToken(token) {
			sawFlag = true
			i++
			continue
		}
		if strings.HasPrefix(token, "-") && token != "-" {
			next, consumed := consumeFlagToken(current.FlagSet, token, args, i)
			if consumed {
				sawFlag = true
				i = next
				continue
			}
			return invocationAnalysis{
				command:      current,
				shape:        shapeForCommand(current, true),
				unknownToken: token,
				unknownFlag:  true,
			}
		}
		if len(current.Subcommands) > 0 {
			return invocationAnalysis{
				command:      current,
				shape:        invocationShapeUnknownChild,
				unknownToken: token,
			}
		}
		return invocationAnalysis{command: current, shape: invocationShapeLeaf}
	}

	return invocationAnalysis{command: current, shape: shapeForCommand(current, sawFlag)}
}

func shapeForCommand(command *ffcli.Command, sawFlag bool) invocationShape {
	if command == nil || len(command.Subcommands) == 0 {
		return invocationShapeLeaf
	}
	if sawFlag {
		return invocationShapeGroupWithFlags
	}
	return invocationShapeBareGroup
}

func shouldRejectUnknownChild(root *ffcli.Command, analysis invocationAnalysis, commandName string) bool {
	if analysis.shape != invocationShapeUnknownChild || analysis.command == nil || analysis.command == root {
		return false
	}

	return !preservesLegacyChild(analysis, commandName)
}

func preservesLegacyChild(analysis invocationAnalysis, commandName string) bool {
	token := strings.TrimSpace(analysis.unknownToken)
	if token == "get" && findDirectSubcommand(analysis.command, "view") != nil {
		return true
	}
	if token == "set" && findDirectSubcommand(analysis.command, "edit") != nil {
		return true
	}

	switch commandName {
	case "asc apps":
		return token == "create"
	case "asc submit":
		return token == "create" || token == "preflight"
	default:
		return false
	}
}

func isHelpToken(token string) bool {
	return token == "-h" || token == "--help" || strings.HasPrefix(token, "--help=")
}

func httpStatusFromError(err error) int {
	var statusError interface{ HTTPStatusCode() int }
	if !errors.As(err, &statusError) {
		return 0
	}
	status := statusError.HTTPStatusCode()
	if status < 400 || status > 599 {
		return 0
	}
	return status
}

func shouldRenderGroupHelp(analysis invocationAnalysis, err error) bool {
	if !errors.Is(err, flag.ErrHelp) || shared.ClassifyUsageError(err) != "" || analysis.command == nil {
		return false
	}
	if analysis.unknownToken != "" || len(analysis.command.Subcommands) == 0 || hasDefinedFlags(analysis.command.FlagSet) {
		return false
	}
	return analysis.shape == invocationShapeBareGroup ||
		analysis.shape == invocationShapeGroupWithFlags
}

func hasDefinedFlags(flagSet *flag.FlagSet) bool {
	if flagSet == nil {
		return false
	}
	found := false
	flagSet.VisitAll(func(*flag.Flag) { found = true })
	return found
}

func printUnknownFlagSuggestion(analysis invocationAnalysis) {
	if !analysis.unknownFlag || analysis.command == nil || analysis.command.FlagSet == nil {
		return
	}
	input := strings.TrimLeft(strings.SplitN(analysis.unknownToken, "=", 2)[0], "-")
	visibleFlags := shared.VisibleHelpFlags(analysis.command.FlagSet)
	candidates := make([]string, 0, len(visibleFlags))
	for _, f := range visibleFlags {
		candidates = append(candidates, f.Name)
	}
	printFlagSuggestions(input, candidates)
}

func printUnknownSubcommandSuggestion(analysis invocationAnalysis) {
	if analysis.shape != invocationShapeUnknownChild || analysis.command == nil || analysis.command.Name == "asc" {
		return
	}
	candidates := make([]string, 0, len(analysis.command.Subcommands))
	for _, sub := range analysis.command.Subcommands {
		candidates = append(candidates, sub.Name)
	}
	printSuggestions(analysis.unknownToken, candidates, "")
}

func printSuggestions(input string, candidates []string, prefix string) {
	suggestions := suggest.Commands(input, candidates)
	printSuggestionList(suggestions, prefix)
}

func printFlagSuggestions(input string, candidates []string) {
	printSuggestionList(suggest.Flags(input, candidates), "--")
}

func printSuggestionList(suggestions []string, prefix string) {
	if len(suggestions) == 0 {
		return
	}
	for i, item := range suggestions {
		suggestions[i] = prefix + shared.SanitizeTerminal(item)
	}
	fmt.Fprintf(os.Stderr, "Did you mean: %s?\n", strings.Join(suggestions, ", "))
}
