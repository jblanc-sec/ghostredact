package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ghostredact/ghostredact/internal/redact"
	"gopkg.in/yaml.v3"
)

type customFile struct {
	Patterns []struct {
		Name  string `json:"name" yaml:"name"`
		Regex string `json:"regex" yaml:"regex"`
	} `json:"patterns" yaml:"patterns"`
}

func main() {
	inPath := flag.String("in", "", "input file path (default: stdin)")
	outPath := flag.String("out", "", "output file path (default: stdout)")
	format := flag.String("format", "text", "input format: text|json")
	mode := flag.String("mode", "mask", "redaction mode: mask|hash|tag")
	salt := flag.String("salt", "", "salt (used when mode=hash)")
	types := flag.String("types", "", "comma-separated detectors to enable (blank = defaults). Known: email,phone,cc,ipv4,ipv6,iban,cpf,cnpj,cep,rg")
	locale := flag.String("locale", "", "optional locale pack (e.g., 'br'). Multiple packs comma-separated.")
	threads := flag.Int("threads", runtime.NumCPU(), "worker threads for text mode")
	reportPath := flag.String("report", "", "optional report file (JSON). Use '-' for stderr")
	customPath := flag.String("custom", "", "optional JSON/YAML file with user-defined regex patterns")

	flag.Parse()

	custom := map[string]string{}
	if strings.TrimSpace(*customPath) != "" {
		cm, err := loadCustom(*customPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "custom patterns error:", err)
			os.Exit(1)
		}
		custom = cm
	}

	cfg := redact.Config{
		Mode:   strings.ToLower(*mode),
		Salt:   *salt,
		Types:  strings.ToLower(*types),
		Locale: strings.ToLower(*locale),
		Custom: custom,
	}
	rd, err := redact.NewRedactor(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	// Open input
	var in io.Reader = os.Stdin
	if *inPath != "" && *inPath != "-" {
		f, err := os.Open(*inPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error opening input:", err)
			os.Exit(1)
		}
		defer f.Close()
		in = f
	}

	// Open output
	var out io.Writer = os.Stdout
	if *outPath != "" && *outPath != "-" {
		f, err := os.Create(*outPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error creating output:", err)
			os.Exit(1)
		}
		defer f.Close()
		out = f
	}

	switch strings.ToLower(*format) {
	case "text":
		processText(in, out, rd, *threads)
	case "json":
		if err := processJSON(in, out, rd); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "unknown -format; use text || json")
		os.Exit(2)
	}

	// Handle report if requested
	if *reportPath != "" {
		if err := writeReport(*reportPath, rd.Counts()); err != nil {
			fmt.Fprintln(os.Stderr, "report error:", err)
			os.Exit(1)
		}
	}
}

func processText(in io.Reader, out io.Writer, rd *redact.Redactor, threads int) {
	// Simple streaming line-by-line redaction.
	sc := bufio.NewScanner(in)
	// Increase buffer for very long lines
	const maxCap = 4 * 1024 * 1024
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, maxCap)

	w := bufio.NewWriter(out)
	defer w.Flush()

	for sc.Scan() {
		line := sc.Text()
		line = rd.RedactString(line)
		fmt.Fprintln(w, line)
	}
	if err := sc.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "scan error:", err)
	}
}

func processJSON(in io.Reader, out io.Writer, rd *redact.Redactor) error {
	var data interface{}
	dec := json.NewDecoder(in)
	dec.UseNumber()
	if err := dec.Decode(&data); err != nil {
		return err
	}
	redactWalk(&data, rd)
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

func redactWalk(v *interface{}, rd *redact.Redactor) {
	switch t := (*v).(type) {
	case map[string]interface{}:
		for k, vv := range t {
			tmp := interface{}(vv)
			redactWalk(&tmp, rd)
			t[k] = tmp
		}
	case []interface{}:
		for i := range t {
			tmp := interface{}(t[i])
			redactWalk(&tmp, rd)
			t[i] = tmp
		}
	case string:
		*v = rd.RedactString(t)
	default:
		// numbers, bools, nil → leave
	}
}

func writeReport(dest string, counts map[string]int) error {
	b, err := json.MarshalIndent(counts, "", "  ")
	if err != nil {
		return err
	}
	if dest == "-" {
		_, err = os.Stderr.Write(append(b, '\n'))
		return err
	}
	return os.WriteFile(dest, b, 0644)
}

func loadCustom(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cf customFile
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		if err := json.Unmarshal(b, &cf); err != nil {
			return nil, err
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(b, &cf); err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unsupported custom file extension (use .json, .yaml, || .yml)")
	}
	out := map[string]string{}
	for _, p := range cf.Patterns {
		name := strings.TrimSpace(p.Name)
		regx := strings.TrimSpace(p.Regex)
		if name == "" || regx == "" {
			continue
		}
		out[name] = regx
	}
	if len(out) == 0 {
		return nil, errors.New("no patterns found in custom file")
	}
	return out, nil
}
