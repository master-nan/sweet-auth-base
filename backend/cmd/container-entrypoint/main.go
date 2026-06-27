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
	cmd := exec.Command("/app/sweet_admin")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Start(); err != nil {
		log.Fatalf("failed to start backend: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
			}
		}
		log.Fatal(err)
	}
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func shouldRunStartupStep(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "0", "false", "no", "n", "off":
		return false
	default:
		return true
	}
}
