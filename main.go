package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
)

const backupDir = "snapshot-backup"

// dryRun keeps restore safe while testing: nothing is installed or changed.
// Pass --apply to do it for real.
var dryRun = true

var dotfiles = []string{".bashrc", ".gitconfig", ".vimrc", ".profile"}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	for _, a := range os.Args[2:] {
		if a == "--apply" {
			dryRun = false
		}
	}

	switch os.Args[1] {
	case "save":
		save()
	case "restore":
		restore()
	case "--help", "-h", "help":
		printUsage()
	case "--version", "-v", "version":
		fmt.Println("snapshot v1.0.0")
	default:
		fmt.Println("unknown command:", os.Args[1])
		printUsage()
	}
}

func printUsage() {
	fmt.Println(`snapshot - save and restore your Ubuntu setup

USAGE:
  snapshot save              capture installed apps and dotfiles
  snapshot restore           preview what would be restored (safe, dry-run)
  snapshot restore --apply   actually reinstall selected apps and dotfiles
  snapshot --help            show this help
  snapshot --version         show version

Backups are stored in the 'snapshot-backup' folder.`)
}

// ---------------- SAVE ----------------

func save() {
	fmt.Println("Saving your setup...")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		fmt.Println("could not create backup folder:", err)
		return
	}
	saveAptPackages()
	saveSnapPackages()
	saveDotfiles()
	fmt.Println("\nDone. Your setup is saved in the '" + backupDir + "' folder.")
}

func saveAptPackages() {
	manual, err := exec.Command("bash", "-c", "apt-mark showmanual").Output()
	if err != nil {
		fmt.Println("  ! could not read apt packages:", err)
		return
	}
	baseline, err := exec.Command("bash", "-c",
		"zcat /var/log/installer/initial-status.gz 2>/dev/null | sed -n 's/^Package: //p' | sort -u").Output()
	if err != nil || len(strings.TrimSpace(string(baseline))) == 0 {
		writeFile(filepath.Join(backupDir, "apt-packages.txt"), manual)
		fmt.Println("  + saved apt list (UNFILTERED - baseline not found)")
		return
	}
	baseSet := make(map[string]bool)
	for _, p := range strings.Fields(string(baseline)) {
		baseSet[p] = true
	}
	var userPkgs []string
	for _, p := range strings.Fields(string(manual)) {
		if !baseSet[p] {
			userPkgs = append(userPkgs, p)
		}
	}
	out := strings.Join(userPkgs, "\n")
	if out != "" {
		out += "\n"
	}
	writeFile(filepath.Join(backupDir, "apt-packages.txt"), []byte(out))
	fmt.Printf("  + saved apt list (%d user-installed packages)\n", len(userPkgs))
}

func saveSnapPackages() {
	out, err := exec.Command("bash", "-c", "snap list | awk 'NR>1 {print $1}'").Output()
	if err != nil {
		fmt.Println("  - skipped snaps (snap not available)")
		return
	}
	writeFile(filepath.Join(backupDir, "snap-packages.txt"), out)
	fmt.Println("  + saved snap package list")
}

func saveDotfiles() {
	home, _ := os.UserHomeDir()
	dfDir := filepath.Join(backupDir, "dotfiles")
	os.MkdirAll(dfDir, 0755)
	for _, name := range dotfiles {
		src := filepath.Join(home, name)
		data, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		writeFile(filepath.Join(dfDir, name), data)
		fmt.Println("  + saved", name)
	}
}

// ---------------- RESTORE ----------------

func restore() {
	if dryRun {
		fmt.Println("DRY RUN - nothing will be installed or changed.")
		fmt.Println("(run with --apply to do it for real)")
	}
	fmt.Println()
	restoreAptPackages()
	restoreSnapPackages()
	restoreDotfiles()
	fmt.Println("\nRestore complete.")
}

func restoreAptPackages() {
	data, err := os.ReadFile(filepath.Join(backupDir, "apt-packages.txt"))
	if err != nil {
		fmt.Println("  - no apt package list found, skipping")
		return
	}
	all := strings.Fields(string(data))
	if len(all) == 0 {
		return
	}
	var chosen []string
	prompt := &survey.MultiSelect{
		Message: "Select apt packages to restore (space to tick, enter to confirm):",
		Options: all,
	}
	if err := survey.AskOne(prompt, &chosen, survey.WithPageSize(15)); err != nil {
		fmt.Println("  selection cancelled, skipping apt")
		return
	}
	if len(chosen) == 0 {
		fmt.Println("  nothing selected, skipping apt")
		return
	}
	if dryRun {
		fmt.Printf("  [dry-run] would install %d apt packages:\n    %s\n",
			len(chosen), strings.Join(chosen, " "))
		return
	}
	args := append([]string{"install", "-y"}, chosen...)
	runMaybeSudo("apt", args...)
}

func restoreSnapPackages() {
	data, err := os.ReadFile(filepath.Join(backupDir, "snap-packages.txt"))
	if err != nil {
		fmt.Println("  - no snap list found, skipping")
		return
	}
	all := strings.Fields(string(data))
	if len(all) == 0 {
		return
	}
	var chosen []string
	prompt := &survey.MultiSelect{
		Message: "Select snaps to restore (space to tick, enter to confirm):",
		Options: all,
	}
	if err := survey.AskOne(prompt, &chosen, survey.WithPageSize(15)); err != nil {
		fmt.Println("  selection cancelled, skipping snaps")
		return
	}
	for _, name := range chosen {
		if dryRun {
			fmt.Println("  [dry-run] would install snap:", name)
			continue
		}
		fmt.Println("  installing snap:", name)
		runMaybeSudo("snap", "install", name)
	}
}

func restoreDotfiles() {
	home, _ := os.UserHomeDir()
	dfDir := filepath.Join(backupDir, "dotfiles")
	entries, err := os.ReadDir(dfDir)
	if err != nil {
		fmt.Println("  - no dotfiles found, skipping")
		return
	}
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(dfDir, e.Name()))
		if err != nil {
			continue
		}
		if dryRun {
			fmt.Println("  [dry-run] would restore", e.Name())
			continue
		}
		dest := filepath.Join(home, e.Name())
		writeFile(dest, data)
		fmt.Println("  + restored", e.Name())
	}
}

// ---------------- helpers ----------------

// runMaybeSudo runs a command, using sudo only when not already root.
func runMaybeSudo(name string, args ...string) {
	if isRoot() {
		runInteractive(name, args...)
	} else {
		runInteractive("sudo", append([]string{name}, args...)...)
	}
}

func isRoot() bool {
	u, err := user.Current()
	if err != nil {
		return false
	}
	return u.Uid == "0"
}

func writeFile(path string, data []byte) {
	if err := os.WriteFile(path, data, 0644); err != nil {
		fmt.Println("  ! error writing", path, ":", err)
	}
}

func confirm(question string) bool {
	fmt.Print(question + " [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}

func runInteractive(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("  ! command failed:", err)
	}
}