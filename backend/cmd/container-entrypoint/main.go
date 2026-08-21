package main

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func main() {
	if shouldRunStartupStep(os.Getenv("APP_RUN_MIGRATIONS")) {
		if err := run("/app/migrate"); err != nil {
			log.Fatalf("migration failed: %v", err)
		}
	} else {
		log.Println("migration skipped because APP_RUN_MIGRATIONS=false")
	}
	if shouldRunStartupStep(os.Getenv("APP_RUN_SEEDS")) {
		if err := run("/app/migrate", "seed"); err != nil {
			log.Fatalf("seed failed: %v", err)
		}
	} else {
		log.Println("seed skipped because APP_RUN_SEEDS=false")
	}
	if err := runStrictPreflight(); err != nil {
		log.Fatalf("database preflight failed: %v", err)
	}
	if err := execApplication("/app/sweet_admin", os.Environ()); err != nil {
		log.Fatalf("failed to exec backend: %v", err)
	}
}

var syscallExec = syscall.Exec

func execApplication(path string, environment []string) error {
	return syscallExec(path, []string{path}, environment)
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func runStrictPreflight() error {
	cmd := exec.Command("/app/db-preflight")
	cmd.Env = environmentWith("APP_DB_PREFLIGHT_REQUIRE_MIGRATED", "true")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func environmentWith(key string, value string) []string {
	prefix := key + "="
	environment := make([]string, 0, len(os.Environ())+1)
	for _, item := range os.Environ() {
		if !strings.HasPrefix(item, prefix) {
			environment = append(environment, item)
		}
	}
	return append(environment, prefix+value)
}

func shouldRunStartupStep(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "0", "false", "no", "n", "off":
		return false
	default:
		return true
	}
}
