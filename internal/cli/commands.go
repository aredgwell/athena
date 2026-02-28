package cli

import (
	"github.com/spf13/cobra"
)

func init() {
	// Top-level commands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(indexCmd)
	rootCmd.AddCommand(gcCmd)
	rootCmd.AddCommand(toolsCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(capabilitiesCmd)
	rootCmd.AddCommand(reportCmd)
	rootCmd.AddCommand(changelogCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(mcpCmd)

	// Grouped commands
	rootCmd.AddCommand(policyCmd)
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(securityCmd)
	rootCmd.AddCommand(contextCmd)
	rootCmd.AddCommand(noteCmd)
	rootCmd.AddCommand(reviewCmd)
	rootCmd.AddCommand(commitCmd)
	rootCmd.AddCommand(releaseCmd)
	rootCmd.AddCommand(hooksCmd)
	rootCmd.AddCommand(optimizeCmd)
}

// --- init ---

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize or scaffold .athena/ and managed files",
	RunE:  runInit,
}

func init() {
	initCmd.Flags().Bool("force", false, "Force overwrite existing files")
	initCmd.Flags().Bool("dry-run", false, "Show what would be done without writing")
	initCmd.Flags().String("preset", "standard", "Preset: minimal, standard, or full")
	initCmd.Flags().Bool("with-pre-commit", false, "Include pre-commit hooks")
}

// --- upgrade ---

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade managed files with checksum comparison",
	RunE:  runUpgrade,
}

func init() {
	upgradeCmd.Flags().Bool("dry-run", false, "Show what would be done without writing")
}

// --- check ---

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate notes, frontmatter, and schema compliance",
	RunE:  runCheck,
}

func init() {
	checkCmd.Flags().Bool("fix", false, "Attempt to fix issues")
	checkCmd.Flags().Bool("strict-schema", false, "Enforce latest schema version")
	checkCmd.Flags().Bool("secrets", false, "Include secrets scan")
	checkCmd.Flags().Bool("workflows", false, "Include workflow lint")
}

// --- index ---

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Rebuild .ai/index.yaml from notes",
	RunE:  runIndex,
}

// --- gc ---

var gcCmd = &cobra.Command{
	Use:   "gc",
	Short: "Mark stale notes for garbage collection",
	RunE:  runGC,
}

func init() {
	gcCmd.Flags().Int("days", 45, "Staleness threshold in days")
	gcCmd.Flags().Bool("dry-run", false, "Show what would be marked without modifying")
}

// --- tools ---

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Check tool availability",
	RunE:  runTools,
}

func init() {
	toolsCmd.Flags().Bool("strict", false, "Treat missing recommended tools as errors")
}

// --- doctor ---

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run repository diagnostics",
	RunE:  runDoctor,
}

// --- capabilities ---

var capabilitiesCmd = &cobra.Command{
	Use:   "capabilities",
	Short: "Print supported commands and schema versions",
	RunE:  runCapabilities,
}

// --- report ---

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Compute memory effectiveness metrics",
	RunE:  runReport,
}

// --- changelog ---

var changelogCmd = &cobra.Command{
	Use:   "changelog",
	Short: "Update CHANGELOG.md from conventional commits",
	RunE:  runChangelog,
}

func init() {
	changelogCmd.Flags().String("since", "", "Baseline tag")
	changelogCmd.Flags().String("next", "", "Next version")
	changelogCmd.Flags().Bool("dry-run", false, "Show what would be generated")
}

// --- completion ---

var completionCmd = &cobra.Command{
	Use:       "completion [bash|zsh|fish|powershell]",
	Short:     "Generate shell completions",
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(cmd.OutOrStdout())
		case "zsh":
			return rootCmd.GenZshCompletion(cmd.OutOrStdout())
		case "fish":
			return rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
		}
		return nil
	},
}

// --- mcp ---

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server over stdio",
	Long: `Start a Model Context Protocol (MCP) server that exposes Athena
commands as tools and resources over stdio transport. Intended for
integration with AI agents (Claude Code, Cursor, etc.).`,
	RunE: runMCP,
}

// --- policy gate ---

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Policy operations",
}

var policyGateCmd = &cobra.Command{
	Use:   "gate",
	Short: "Evaluate PR/revision against policy gates",
	RunE:  runPolicyGate,
}

func init() {
	policyCmd.AddCommand(policyGateCmd)
	policyGateCmd.Flags().String("pr", "", "Target ref to evaluate")
}

// --- plan / apply / rollback ---

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Compute a deterministic mutation plan",
	RunE:  runPlan,
}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Execute a stored plan",
	RunE:  runApply,
}

func init() {
	applyCmd.Flags().String("plan-id", "", "Plan ID to apply")
}

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Revert a transaction",
	RunE:  runRollback,
}

func init() {
	rollbackCmd.Flags().String("tx", "", "Transaction ID")
	rollbackCmd.Flags().Int("to-step", 0, "Roll back to step N")
}

// --- security scan ---

var securityCmd = &cobra.Command{
	Use:   "security",
	Short: "Security operations",
}

var securityScanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Run security scans (gitleaks/actionlint)",
	RunE:  runSecurityScan,
}

func init() {
	securityCmd.AddCommand(securityScanCmd)
	securityScanCmd.Flags().Bool("secrets", false, "Run secrets scan only")
	securityScanCmd.Flags().Bool("workflows", false, "Run workflow lint only")
	securityScanCmd.Flags().String("report-format", "json", "Report format: json or sarif")
}

// --- context pack / mcp / budget ---

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Context packing operations",
}

var contextPackCmd = &cobra.Command{
	Use:   "pack",
	Short: "Pack repository context via repomix",
	RunE:  runContextPack,
}

var contextMCPCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start or validate repomix MCP mode",
	RunE:  runContextMCP,
}

var contextBudgetCmd = &cobra.Command{
	Use:   "budget",
	Short: "Estimate context token budget",
	RunE:  runContextBudget,
}

func init() {
	contextCmd.AddCommand(contextPackCmd)
	contextCmd.AddCommand(contextMCPCmd)
	contextCmd.AddCommand(contextBudgetCmd)

	contextPackCmd.Flags().String("profile", "", "Context profile: review, handoff, or release")
	contextPackCmd.Flags().Bool("changed", false, "Pack only changed files")
	contextPackCmd.Flags().Bool("stdout", false, "Stream to stdout")
	contextPackCmd.Flags().String("output", "", "Output path")
	contextPackCmd.Flags().Bool("dry-run", false, "Show what would be packed")

	contextMCPCmd.Flags().Bool("stdio", false, "Use stdio transport")

	contextBudgetCmd.Flags().String("profile", "", "Profile to estimate")
	contextBudgetCmd.Flags().Int("max-tokens", 0, "Maximum token budget")
}

// --- note new / close / promote / list ---

var noteCmd = &cobra.Command{
	Use:   "note",
	Short: "Note lifecycle operations",
}

var noteNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new note from template",
	RunE:  runNoteNew,
}

var noteCloseCmd = &cobra.Command{
	Use:   "close [path]",
	Short: "Transition a note's status",
	Args:  cobra.ExactArgs(1),
	RunE:  runNoteClose,
}

var notePromoteCmd = &cobra.Command{
	Use:   "promote [path]",
	Short: "Mark a note as promoted",
	Args:  cobra.ExactArgs(1),
	RunE:  runNotePromote,
}

var noteListCmd = &cobra.Command{
	Use:   "list",
	Short: "List notes with optional filters",
	RunE:  runNoteList,
}

func init() {
	noteCmd.AddCommand(noteNewCmd)
	noteCmd.AddCommand(noteCloseCmd)
	noteCmd.AddCommand(notePromoteCmd)
	noteCmd.AddCommand(noteListCmd)

	noteNewCmd.Flags().String("type", "", "Note type")
	noteNewCmd.Flags().String("slug", "", "Note slug")
	noteNewCmd.Flags().String("title", "", "Note title")

	noteCloseCmd.Flags().String("status", "", "Target status")

	notePromoteCmd.Flags().String("target", "", "Promotion target path")

	noteListCmd.Flags().String("status", "", "Filter by status")
	noteListCmd.Flags().String("type", "", "Filter by type")
}

// --- review promotions / weekly ---

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Review operations",
}

var reviewPromotionsCmd = &cobra.Command{
	Use:   "promotions",
	Short: "List promotion-ready notes",
	RunE:  runReviewPromotions,
}

var reviewWeeklyCmd = &cobra.Command{
	Use:   "weekly",
	Short: "Run weekly review (gc + promotions + check)",
	RunE:  runReviewWeekly,
}

func init() {
	reviewCmd.AddCommand(reviewPromotionsCmd)
	reviewCmd.AddCommand(reviewWeeklyCmd)
	reviewWeeklyCmd.Flags().Int("days", 7, "Review window in days")
}

// --- commit lint ---

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Commit operations",
}

var commitLintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Validate commit messages against Conventional Commits",
	RunE:  runCommitLint,
}

func init() {
	commitCmd.AddCommand(commitLintCmd)
	commitLintCmd.Flags().String("from", "", "Start ref")
	commitLintCmd.Flags().String("to", "", "End ref")
}

// --- release propose / approve ---

var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "Release operations",
}

var releaseProposeCmd = &cobra.Command{
	Use:   "propose",
	Short: "Generate a release proposal with gate checks",
	RunE:  runReleasePropose,
}

var releaseApproveCmd = &cobra.Command{
	Use:   "approve",
	Short: "Approve and execute a release proposal",
	RunE:  runReleaseApprove,
}

func init() {
	releaseCmd.AddCommand(releaseProposeCmd)
	releaseCmd.AddCommand(releaseApproveCmd)

	releaseProposeCmd.Flags().String("since", "", "Baseline tag")
	releaseProposeCmd.Flags().String("next", "", "Next version")

	releaseApproveCmd.Flags().String("proposal-id", "", "Proposal ID to approve")
}

// --- hooks install ---

var hooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Hook operations",
}

var hooksInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install or update pre-commit hooks",
	RunE:  runHooksInstall,
}

func init() {
	hooksCmd.AddCommand(hooksInstallCmd)
	hooksInstallCmd.Flags().Bool("pre-commit", false, "Install pre-commit hooks")
}

// --- optimize recommend ---

var optimizeCmd = &cobra.Command{
	Use:   "optimize",
	Short: "Optimization operations",
}

var optimizeRecommendCmd = &cobra.Command{
	Use:   "recommend",
	Short: "Propose bounded tuning changes from telemetry",
	RunE:  runOptimizeRecommend,
}

func init() {
	optimizeCmd.AddCommand(optimizeRecommendCmd)
	optimizeRecommendCmd.Flags().String("window", "30d", "Analysis window")
}
