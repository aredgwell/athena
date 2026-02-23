package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/amr-athena/athena/internal/notes"
	"github.com/spf13/cobra"
)

func runNoteNew(cmd *cobra.Command, args []string) error {
	start := time.Now()
	noteType, _ := cmd.Flags().GetString("type")
	slug, _ := cmd.Flags().GetString("slug")
	title, _ := cmd.Flags().GetString("title")

	note, err := notes.NewNote(aiDir(), noteType, slug, title)
	if err != nil {
		return err
	}

	env := NewEnvelope("note new", time.Since(start)).WithData(map[string]any{
		"path": note.Path,
		"id":   note.Frontmatter.ID,
	})
	writeOutput(cmd, env, func(w io.Writer) {
		fmt.Fprintf(w, "Created %s\n", note.Path)
	})
	return nil
}

func runNoteClose(cmd *cobra.Command, args []string) error {
	start := time.Now()
	if len(args) == 0 {
		return fmt.Errorf("note path required as argument")
	}
	status, _ := cmd.Flags().GetString("status")
	if status == "" {
		status = "closed"
	}

	if err := notes.CloseNote(args[0], status); err != nil {
		return err
	}

	env := NewEnvelope("note close", time.Since(start)).WithData(map[string]any{
		"path":   args[0],
		"status": status,
	})
	writeOutput(cmd, env, func(w io.Writer) {
		fmt.Fprintf(w, "Closed %s → %s\n", args[0], status)
	})
	return nil
}

func runNotePromote(cmd *cobra.Command, args []string) error {
	start := time.Now()
	if len(args) == 0 {
		return fmt.Errorf("note path required as argument")
	}
	target, _ := cmd.Flags().GetString("target")

	if err := notes.PromoteNote(args[0], target); err != nil {
		return err
	}

	env := NewEnvelope("note promote", time.Since(start)).WithData(map[string]any{
		"path":   args[0],
		"target": target,
	})
	writeOutput(cmd, env, func(w io.Writer) {
		fmt.Fprintf(w, "Promoted %s → %s\n", args[0], target)
	})
	return nil
}

func runNoteList(cmd *cobra.Command, args []string) error {
	start := time.Now()
	statusFilter, _ := cmd.Flags().GetString("status")
	typeFilter, _ := cmd.Flags().GetString("type")

	allNotes, err := notes.ListNotes(aiDir(), statusFilter, typeFilter)
	if err != nil {
		return err
	}

	type noteSummary struct {
		Path   string `json:"path"`
		ID     string `json:"id"`
		Type   string `json:"type"`
		Status string `json:"status"`
		Title  string `json:"title"`
	}
	var summaries []noteSummary
	for _, n := range allNotes {
		summaries = append(summaries, noteSummary{
			Path:   n.Path,
			ID:     n.Frontmatter.ID,
			Type:   n.Frontmatter.Type,
			Status: n.Frontmatter.Status,
			Title:  n.Frontmatter.Title,
		})
	}

	env := NewEnvelope("note list", time.Since(start)).WithData(map[string]any{
		"count": len(summaries),
		"notes": summaries,
	})
	writeOutput(cmd, env, func(w io.Writer) {
		fmt.Fprintf(w, "%d notes\n", len(summaries))
		for _, s := range summaries {
			fmt.Fprintf(w, "  [%s] %s (%s) %s\n", s.Status, s.ID, s.Type, s.Title)
		}
	})
	return nil
}
