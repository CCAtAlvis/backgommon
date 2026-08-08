// Command backgommon is the CLI for the insights platform and static serve.
//
// register / run / sweep / ingest / status talk to the local catalog under
// --data-dir (default ~/.backgommon). serve is only required for the web UI.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/CCAtAlvis/backgommon/pkg/catalog"
	"github.com/CCAtAlvis/backgommon/pkg/daemon"
	"github.com/CCAtAlvis/backgommon/pkg/jobs"
	"github.com/CCAtAlvis/backgommon/pkg/serve"
	"gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "serve":
		os.Exit(cmdServe(os.Args[2:]))
	case "register":
		os.Exit(cmdRegister(os.Args[2:]))
	case "unregister":
		os.Exit(cmdUnregister(os.Args[2:]))
	case "run":
		os.Exit(cmdRun(os.Args[2:]))
	case "sweep":
		os.Exit(cmdSweep(os.Args[2:]))
	case "ingest":
		os.Exit(cmdIngest(os.Args[2:]))
	case "status", "jobs":
		os.Exit(cmdJobs(os.Args[2:]))
	case "open":
		os.Exit(cmdOpen(os.Args[2:]))
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `backgommon — insights platform CLI

Usage:
  backgommon serve [flags] [dir]
  backgommon register <binary> [--name NAME]
  backgommon unregister <name>
  backgommon run <name> [--config FILE] [--set k=v]
  backgommon sweep <name> [--config FILE] [--cases FILE] [--axes FILE]
  backgommon ingest <dir> [--strategy NAME]
  backgommon status | jobs
  backgommon open

Commands talk to the local catalog (--data-dir, default ~/.backgommon).
serve is only needed for the web UI (or static folder viewing).

Commands:
  serve       Web UI + API (no dir), or static folder serve (with dir)
  register    Describe binary and add to catalog (--name overrides describe.name)
  unregister  Remove strategy from catalog
  run         Run a single backtest (sync)
  sweep       Run a parameter sweep (sync)
  ingest      Import an existing run/sweep folder
  status      List recent jobs
  open        Open dashboard in browser (requires serve)
`)
}

func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".backgommon"
	}
	return filepath.Join(home, ".backgommon")
}

func dataDirFlag(fs *flag.FlagSet) *string {
	return fs.String("data-dir", defaultDataDir(), "catalog data directory")
}

func openCat(dataDir string) (*catalog.Catalog, error) {
	return catalog.Open(dataDir)
}

func cmdServe(args []string) int {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", "127.0.0.1:8743", "listen address")
	dataDir := dataDirFlag(fs)
	_ = fs.Parse(reorderFlags(args))

	if fs.NArg() > 0 {
		dir := fs.Arg(0)
		abs, err := filepath.Abs(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "serve: %v\n", err)
			return 1
		}
		st, err := os.Stat(abs)
		if err != nil || !st.IsDir() {
			fmt.Fprintf(os.Stderr, "serve: %s is not a directory\n", abs)
			return 1
		}
		if err := serve.ListenAndServe(*addr, abs); err != nil {
			fmt.Fprintf(os.Stderr, "serve: %v\n", err)
			return 1
		}
		return 0
	}

	cat, err := openCat(*dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "serve: %v\n", err)
		return 1
	}
	defer cat.Close()
	srv := daemon.New(cat, *addr)
	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "serve: %v\n", err)
		return 1
	}
	return 0
}

func cmdRegister(args []string) int {
	fs := flag.NewFlagSet("register", flag.ExitOnError)
	name := fs.String("name", "", "override catalog name (default: binary describe.name)")
	dataDir := dataDirFlag(fs)
	_ = fs.Parse(reorderFlags(args))
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: backgommon register <binary> [--name NAME]")
		return 2
	}
	cat, err := openCat(*dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "register: %v\n", err)
		return 1
	}
	defer cat.Close()
	res, err := jobs.Register(cat, fs.Arg(0), *name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "register: %v\n", err)
		return 1
	}
	fmt.Printf("Registered %s @ %s\n", res.Strategy.Name, res.Strategy.Version)
	fmt.Printf("  path: %s\n", res.Strategy.AbsPath)
	if res.Updated {
		fmt.Println("  (updated existing registration)")
	}
	fmt.Printf("  data: %s\n", *dataDir)
	return 0
}

func cmdUnregister(args []string) int {
	fs := flag.NewFlagSet("unregister", flag.ExitOnError)
	dataDir := dataDirFlag(fs)
	_ = fs.Parse(reorderFlags(args))
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: backgommon unregister <name>")
		return 2
	}
	cat, err := openCat(*dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unregister: %v\n", err)
		return 1
	}
	defer cat.Close()
	if err := cat.Unregister(fs.Arg(0)); err != nil {
		fmt.Fprintf(os.Stderr, "unregister: %v\n", err)
		return 1
	}
	fmt.Printf("Unregistered %s\n", fs.Arg(0))
	return 0
}

func cmdRun(args []string) int {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	dataDir := dataDirFlag(fs)
	configPath := fs.String("config", "", "YAML/JSON config file")
	clean, sets := extractSets(args)
	_ = fs.Parse(reorderFlags(clean))
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: backgommon run <name> [--config FILE] [--set k=v]")
		return 2
	}
	cfg, err := loadConfigMap(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "run: %v\n", err)
		return 1
	}
	applySetList(cfg, sets)
	cfgBytes, _ := json.Marshal(cfg)

	cat, err := openCat(*dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "run: %v\n", err)
		return 1
	}
	defer cat.Close()
	jr := &jobs.Runner{Cat: cat}
	jobID, err := jr.QueueRun(fs.Arg(0), cfgBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "run: %v\n", err)
		return 1
	}
	j, _ := cat.GetJob(jobID)
	fmt.Printf("Job #%d %s\n", jobID, j.Status)
	if j.ReportDir != "" {
		fmt.Printf("  report: %s\n", j.ReportDir)
	}
	if j.Error != "" {
		fmt.Fprintf(os.Stderr, "  error: %s\n", j.Error)
		return 1
	}
	return 0
}

func cmdSweep(args []string) int {
	fs := flag.NewFlagSet("sweep", flag.ExitOnError)
	dataDir := dataDirFlag(fs)
	configPath := fs.String("config", "", "YAML/JSON base config")
	casesPath := fs.String("cases", "", "YAML/JSON array of case configs")
	axesPath := fs.String("axes", "", "YAML/JSON axes (passed through to worker)")
	workers := fs.Int("workers", 0, "worker count hint")
	sortBy := fs.String("sort-by", "sharpe_ratio", "sort metric")
	clean, sets := extractSets(args)
	_ = fs.Parse(reorderFlags(clean))
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: backgommon sweep <name> [--config FILE] [--cases FILE]")
		return 2
	}
	base, err := loadConfigMap(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sweep: %v\n", err)
		return 1
	}
	applySetList(base, sets)
	baseBytes, _ := json.Marshal(base)

	var casesBytes, axesBytes json.RawMessage
	if *casesPath != "" {
		var cases any
		if err := loadYAMLOrJSON(*casesPath, &cases); err != nil {
			fmt.Fprintf(os.Stderr, "sweep: cases: %v\n", err)
			return 1
		}
		casesBytes, _ = json.Marshal(cases)
	}
	if *axesPath != "" {
		var axes any
		if err := loadYAMLOrJSON(*axesPath, &axes); err != nil {
			fmt.Fprintf(os.Stderr, "sweep: axes: %v\n", err)
			return 1
		}
		axesBytes, _ = json.Marshal(axes)
	}

	cat, err := openCat(*dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sweep: %v\n", err)
		return 1
	}
	defer cat.Close()
	jr := &jobs.Runner{Cat: cat}
	jobID, err := jr.QueueSweep(fs.Arg(0), baseBytes, casesBytes, axesBytes, *workers, *sortBy)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sweep: %v\n", err)
		return 1
	}
	j, _ := cat.GetJob(jobID)
	fmt.Printf("Job #%d %s\n", jobID, j.Status)
	if j.ReportDir != "" {
		fmt.Printf("  sweep: %s\n", j.ReportDir)
	}
	if j.Error != "" {
		fmt.Fprintf(os.Stderr, "  error: %s\n", j.Error)
		return 1
	}
	return 0
}

func cmdIngest(args []string) int {
	fs := flag.NewFlagSet("ingest", flag.ExitOnError)
	dataDir := dataDirFlag(fs)
	strategy := fs.String("strategy", "", "strategy name tag")
	_ = fs.Parse(reorderFlags(args))
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: backgommon ingest <dir>")
		return 2
	}
	cat, err := openCat(*dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ingest: %v\n", err)
		return 1
	}
	defer cat.Close()
	art, err := cat.IngestDir(fs.Arg(0), *strategy, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ingest: %v\n", err)
		return 1
	}
	fmt.Printf("Ingested %s #%d (%s)\n", art.Kind, art.ID, art.Title)
	fmt.Printf("  path: %s\n", art.Path)
	return 0
}

func cmdJobs(args []string) int {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	dataDir := dataDirFlag(fs)
	_ = fs.Parse(reorderFlags(args))
	cat, err := openCat(*dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "status: %v\n", err)
		return 1
	}
	defer cat.Close()
	list, err := cat.ListJobs(50)
	if err != nil {
		fmt.Fprintf(os.Stderr, "status: %v\n", err)
		return 1
	}
	for _, j := range list {
		fmt.Printf("#%d  %-6s  %-16s  %-10s  %s\n", j.ID, j.Type, j.StrategyName, j.Status, j.ReportDir)
		if j.Error != "" {
			fmt.Printf("     error: %s\n", j.Error)
		}
	}
	if len(list) == 0 {
		fmt.Println("(no jobs)")
	}
	return 0
}

func cmdOpen(args []string) int {
	fs := flag.NewFlagSet("open", flag.ExitOnError)
	addr := fs.String("addr", "127.0.0.1:8743", "daemon address")
	_ = fs.Parse(reorderFlags(args))
	url := "http://" + *addr + "/"
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n  url: %s\n", err, url)
		return 1
	}
	fmt.Println(url)
	return 0
}

func reorderFlags(args []string) []string {
	var flags, positionals []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}
		if strings.HasPrefix(a, "-") && a != "-" {
			flags = append(flags, a)
			if !strings.Contains(a, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				name := strings.TrimLeft(a, "-")
				if name != "help" && name != "h" {
					flags = append(flags, args[i+1])
					i++
				}
			}
			continue
		}
		positionals = append(positionals, a)
	}
	return append(flags, positionals...)
}

func extractSets(args []string) (clean []string, sets []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--set" && i+1 < len(args) {
			sets = append(sets, args[i+1])
			i++
			continue
		}
		if strings.HasPrefix(a, "--set=") {
			sets = append(sets, strings.TrimPrefix(a, "--set="))
			continue
		}
		clean = append(clean, a)
	}
	return clean, sets
}

func applySetList(cfg map[string]any, sets []string) {
	for _, kv := range sets {
		k, v, ok := strings.Cut(kv, "=")
		if ok {
			cfg[k] = coerce(v)
		}
	}
}

func loadConfigMap(path string) (map[string]any, error) {
	cfg := map[string]any{}
	if path == "" {
		return cfg, nil
	}
	if err := loadYAMLOrJSON(path, &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func loadYAMLOrJSON(path string, out any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		return yaml.Unmarshal(b, out)
	case ".json":
		return json.Unmarshal(b, out)
	default:
		if err := yaml.Unmarshal(b, out); err == nil {
			return nil
		}
		return json.Unmarshal(b, out)
	}
}

func coerce(v string) any {
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err == nil && fmt.Sprintf("%d", n) == v {
		return n
	}
	var f float64
	if _, err := fmt.Sscanf(v, "%f", &f); err == nil {
		return f
	}
	if v == "true" {
		return true
	}
	if v == "false" {
		return false
	}
	return v
}
