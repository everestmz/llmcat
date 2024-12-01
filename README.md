# llmcat

Generate outlines and partial representations of code repos in a format that's easy for LLMs to consume (and fits inside context windows!).

The RepoMap feature in tools like [Aider](https://github.com/Aider-AI/aider) and [Codestory's Aide](https://github.com/codestoryai) is _awesome_. I wanted a more reusable and portable (think [Unix philosophy](http://www.catb.org/esr/writings/taoup/html/ch01s06.html)) version that I could include in my own agents and projects like [sage](https://github.com/everestmz/sage).

The result is `llmcat` - a tool useful for agentic workflows where an LLM starts with a high-level view of a repository, and expands specific functions until it comes to a solution. It also works great for copying & pasting repos into ChatGPT and Claude.

## Features

- Produce high-level "maps" of repos for LLMs
- Incrementally specify functions to add to the map
- Line numbers and markdown formatting by default
- Smart pagination for large files
- Flexible file filtering with glob patterns
- Built-in ignores for common patterns (like .git directories)
- Fully customizable rendering
- Use as a CLI tool or a library

## Installation

Grab a binary from [the latest release](https://github.com/everestmz/llmcat/releases).

Via Go:

```console
go install github.com/everestmz/llmcat/cmd/llmcat@latest
```

Or clone and build from source:

```console
git clone https://github.com/everestmz/llmcat.git
cd llmcat
make install
```

## Usage

### Basic Usage

Display a single file:
```bash
llmcat llmcat.go
```

<details>
<summary>Output</summary>
<br>

````
```llmcat.go (Lines 1-185 of 185)
1   | package llmcat
2   |
3   | import (
4   |   "fmt"
5   |   "io/fs"
6   |   "os"
7   |   "path/filepath"
8   |   "strings"
9   |
10  |   "github.com/gobwas/glob"
11  | )
12  |
13  | type RenderFileOptions struct {
14  |   OutputMarkdown  bool   `json:"output_markdown"`
15  |   ShowLineNumbers bool   `json:"hide_line_numbers"`
16  |   GutterSeparator string `json:"gutter_separator"`
17  |   PageSize        int    `json:"page_size"`
18  |   StartLine       int    `json:"start_line"`
19  |   ShowPageInfo    bool   `json:"show_page_info"`
20  | }
21  |
22  | func (ro *RenderFileOptions) SetDefaults() {
23  |   if ro.GutterSeparator == "" {
24  |           ro.GutterSeparator = "|"
25  |   }
26  |
27  |   if ro.PageSize == 0 {
28  |           ro.PageSize = 10000
29  |   }
30  |
31  |   if ro.StartLine < 1 {
32  |           ro.StartLine = 1
33  |   }
34  | }
35  |
36  | func RenderFile(filename, text string, options *RenderFileOptions) string {
37  |   outputLines := []string{}
38  |
39  |   options.SetDefaults()
40  |
41  |   lines := strings.Split(text, "\n")
42  |   totalLines := len(lines)
43  |
44  |   gutterWidth := len(fmt.Sprint(len(lines))) + 1 // add 1 line for a space before the separator
45  |
46  |   // Calculate page bounds
47  |   startIndex := options.StartLine - 1
48  |   endIndex := totalLines
49  |
50  |   if options.PageSize > 0 {
51  |           endIndex = startIndex + options.PageSize
52  |           if endIndex > totalLines {
53  |                   endIndex = totalLines
54  |           }
55  |   }
56  |
57  |   // Validate bounds
58  |   if startIndex >= totalLines {
59  |           startIndex = totalLines - 1
60  |           if startIndex < 0 {
61  |                   startIndex = 0
62  |           }
63  |           endIndex = totalLines
64  |   }
65  |
66  |   if options.OutputMarkdown {
67  |           header := fmt.Sprintf("```%s", filename)
68  |           if options.ShowPageInfo && options.PageSize > 0 {
69  |                   header += fmt.Sprintf(" (Lines %d-%d of %d)", startIndex+1, endIndex, totalLines)
70  |           }
71  |           outputLines = append(outputLines, header)
72  |   }
73  |
74  |   if startIndex > 0 {
75  |           marker := fmt.Sprintf("... (%d lines above) ...", startIndex)
76  |           if options.ShowLineNumbers {
77  |                   marker = fmt.Sprintf("%s%s %s", strings.Repeat(" ", gutterWidth), options.GutterSeparator, marke
78  |           }
79  |           outputLines = append(outputLines, marker)
80  |   }
81  |
82  |   for i, line := range lines[startIndex:endIndex] {
83  |           if options.ShowLineNumbers {
84  |                   lineNum := i + startIndex + 1
85  |                   padding := strings.Repeat(" ", gutterWidth-len(fmt.Sprint(lineNum)))
86  |                   line = fmt.Sprintf("%d%s%s %s", lineNum, padding, options.GutterSeparator, line)
87  |           }
88  |
89  |           outputLines = append(outputLines, line)
90  |   }
91  |
92  |   if endIndex < totalLines {
93  |           marker := fmt.Sprintf("... (%d lines below) ...", totalLines-endIndex)
94  |           if options.ShowLineNumbers {
95  |                   marker = fmt.Sprintf("%s%s %s", strings.Repeat(" ", gutterWidth), options.GutterSeparator, marke
96  |           }
97  |           outputLines = append(outputLines, marker)
98  |   }
99  |
100 |   if options.OutputMarkdown {
101 |           outputLines = append(outputLines, "```")
102 |   }
103 |
104 |   return strings.Join(outputLines, "\n")
105 | }
106 |
107 | // We should probably allow for glob-based ignores, extension-based ignores, and some other dir-based filters
108 | type RenderDirectoryOptions struct {
109 |   FileOptions   *RenderFileOptions `json:"file_options"`
110 |   IgnoreGlobs   []string           `json:"ignore_globs"`
111 |   compiledGlobs []glob.Glob
112 | }
113 |
114 | func (rdo *RenderDirectoryOptions) SetDefaults() error {
115 |   rdo.FileOptions.SetDefaults()
116 |
117 |   rdo.IgnoreGlobs = append(rdo.IgnoreGlobs, "**/.git/**")
118 |
119 |   for _, ignoreGlob := range rdo.IgnoreGlobs {
120 |           g, err := glob.Compile(ignoreGlob)
121 |           if err != nil {
122 |                   return err
123 |           }
124 |
125 |           rdo.compiledGlobs = append(rdo.compiledGlobs, g)
126 |   }
127 |
128 |   return nil
129 | }
130 |
131 | func RenderDirectory(dirName string, options *RenderDirectoryOptions) (string, error) {
132 |   var files []string
133 |
134 |   err := options.SetDefaults()
135 |   if err != nil {
136 |           return "", err
137 |   }
138 |
139 |   dirName, err = filepath.Abs(dirName)
140 |   if err != nil {
141 |           return "", err
142 |   }
143 |
144 |   err = filepath.WalkDir(dirName, func(path string, d fs.DirEntry, err error) error {
145 |           for _, ignoreGlob := range options.compiledGlobs {
146 |                   if ignoreGlob.Match(path) {
147 |                           return nil
148 |                   }
149 |           }
150 |
151 |           info, err := d.Info()
152 |           if err != nil {
153 |                   return err
154 |           }
155 |
156 |           if info.IsDir() {
157 |                   return nil
158 |           }
159 |
160 |           // Check if file has execute permission using file mode bits
161 |           if info.Mode()&0111 != 0 {
162 |                   return nil
163 |           }
164 |
165 |           text, err := os.ReadFile(path)
166 |           if err != nil {
167 |                   return err
168 |           }
169 |
170 |           relPath, err := filepath.Rel(dirName, path)
171 |           if err != nil {
172 |                   return err
173 |           }
174 |           files = append(files, RenderFile(relPath, string(text), options.FileOptions))
175 |
176 |           return nil
177 |   })
178 |
179 |   if err != nil {
180 |           return "", err
181 |   }
182 |
183 |   return strings.Join(files, "\n\n"), nil
184 | }
185 |
```
````
</details>

Display an entire directory:
```bash
llmcat .
```

Display a map of a repo or file:
```bash
llmcat --outline .
```

Display a map of the repo, but with the `RenderDirectory` function expanded:
```bash
llmcat --outline --expand "llmcat.go RenderDirectory" .
```

### Navigation

View specific portions of large files:
```bash
# Show first 50 lines
llmcat index.ts --page-size 50

# Start from line 100
llmcat index.ts --page-size 50 --start-line 100
```

### Directory Filtering

Exclude specific files or directories:
```bash
# Ignore specific patterns
llmcat . --ignore "**/test/**" --ignore "**/*.tmp"

# Exclude file extensions
llmcat . --exclude-ext "log,tmp,cache"
```

### Customization

Adjust the output format:
```bash
# Change the gutter separator
llmcat main.go --separator ":"

# Disable line numbers
llmcat main.go --line-numbers=false

# Disable markdown formatting
llmcat main.go --markdown=false
```
