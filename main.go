package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type CodeBlock struct {
	Language  string
	Content   string
	StartLine int
	EndLine   int
}

func extractCodeBlocks(md string) []CodeBlock {
	var blocks []CodeBlock
	parser := goldmark.DefaultParser()
	source := []byte(md)
	node := parser.Parse(text.NewReader(source))

	var extract func(ast.Node)
	extract = func(n ast.Node) {
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if codeBlock, ok := c.(*ast.FencedCodeBlock); ok {
				lang := string(codeBlock.Language(source))
				var contentBuilder strings.Builder
				if codeBlock.Lines().Len() > 0 {
					startLine := codeBlock.Lines().At(0).Start + 1
					endLine := codeBlock.Lines().At(codeBlock.Lines().Len() - 1).Stop + 1

					for i := 0; i < codeBlock.Lines().Len(); i++ {
						line := codeBlock.Lines().At(i)
						contentBuilder.Write(line.Value(source))
						contentBuilder.WriteString("\n")
					}

					blocks = append(blocks, CodeBlock{
						Language:  lang,
						Content:   contentBuilder.String(),
						StartLine: startLine,
						EndLine:   endLine,
					})
				}
			}
			extract(c)
		}
	}
	extract(node)
	return blocks
}

func presentChoices(blocks []CodeBlock, sourceName string) int {
	fmt.Println("Select a code block to execute:")
	for i, block := range blocks {
		lines := strings.Split(block.Content, "\n")
		preview := strings.Join(lines[:min(3, len(lines))], "\n")
		if len(lines) > 3 {
			preview += "\n..."
		}
		fmt.Printf("[%d] %s:%d:%d (%s)\n%s\n\n", i+1, sourceName, block.StartLine, block.EndLine, block.Language, preview)
	}

	var choice int
	for {
		fmt.Print("Enter the number of the code block: ")
		_, err := fmt.Scan(&choice)
		if err == nil && choice > 0 && choice <= len(blocks) {
			break
		}
		fmt.Println("Invalid selection, try again.")
	}
	return choice - 1
}

func executeBlock(block CodeBlock) {
	var shell string
	switch block.Language {
	case "bash", "sh":
		shell = "bash"
	case "fish":
		shell = "fish"
	case "nushell", "nu":
		shell = "nu"
	default:
		fmt.Println("Unsupported language:", block.Language)
		return
	}

	cmd := exec.Command(shell)
	cmd.Stdin = bytes.NewBufferString(block.Content)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("Error executing code block:", err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	var input string
	var sourceName string

	if len(os.Args) > 1 {
		filename := os.Args[1]
		content, err := os.ReadFile(filename)
		if err != nil {
			fmt.Println("Error reading file:", err)
			os.Exit(1)
		}
		input = string(content)
		sourceName = filename
	} else {
		var buf bytes.Buffer
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			buf.WriteString(scanner.Text() + "\n")
		}
		if err := scanner.Err(); err != nil && err != io.EOF {
			fmt.Println("Error reading stdin:", err)
			os.Exit(1)
		}
		input = buf.String()
		sourceName = "stdin"
	}

	blocks := extractCodeBlocks(input)
	if len(blocks) == 0 {
		fmt.Println("No code blocks found.")
		os.Exit(1)
	}

	selectedIndex := presentChoices(blocks, sourceName)
	executeBlock(blocks[selectedIndex])
}
