// scope - a self-profiling search engine
//
// started as a ripgrep clone, evolved into something that watches itself
// while it works. every search shows you which workers did what, how fast,
// and whether the work was evenly distributed.
//
// the whole project lives in two places:
//
//	cmd/      - CLI layer (thin wrappers around the real logic)
//	internal/ - where everything actually happens
package main

import "github.com/Viswesh-G/scope/cmd"

func main() { cmd.Execute() }
