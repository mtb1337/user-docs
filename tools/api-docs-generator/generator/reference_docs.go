package generator

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/snyk/user-docs/tools/api-docs-generator/config"
)

type operationPath struct {
	operation *openapi3.Operation
	pathItem  *openapi3.PathItem
	pathURL   string
	specPath  string
	method    string
	docsHint  string
}

func GenerateReferenceDocs(cfg config.Config, docsBasePath string) error {
	aggregatedDocs, err := aggregateSpecs(cfg, docsBasePath)
	if err != nil {
		return err
	}

	summary := make([]string, len(aggregatedDocs))
	for label, operations := range aggregatedDocs {
		destinationPath := path.Join(docsBasePath, cfg.Output.APIReferencePath, labelToFileName(label))
		summary = append(summary, fmt.Sprintf("* [%s](%s)\n", label, path.Join(cfg.Output.APIReferencePath, labelToFileName(label))))

		err := renderReferenceDocsPage(destinationPath, label, docsBasePath, operations)
		if err != nil {
			return err
		}
	}
	sort.Strings(summary)
	fmt.Printf("generated menu for summary:\n")
	fmt.Printf("%s", strings.Join(summary, ""))

	return nil
}

func aggregateSpecs(cfg config.Config, docsBasePath string) (map[string][]operationPath, error) {
	aggregatedDocs := map[string][]operationPath{}

	for _, spec := range cfg.Specs {
		loader := openapi3.NewLoader()
		doc, err := loader.LoadFromFile(path.Join(docsBasePath, spec.Path))
		if err != nil {
			return nil, err
		}
		for pathURL, pathItem := range doc.Paths.Map() {
			for method, operation := range pathItem.Operations() {
				for _, tag := range operation.Tags {
					tag += spec.Suffix
					aggregatedDocs[tag] = append(aggregatedDocs[tag], operationPath{
						operation: operation,
						pathItem:  pathItem,
						pathURL:   pathURL,
						specPath:  spec.Path,
						method:    method,
						docsHint:  spec.DocsHint,
					})
				}
			}
		}
	}
	return aggregatedDocs, nil
}

func renderReferenceDocsPage(filePath, label, docsPath string, operation []operationPath) error {
	docsFile, err := os.Create(filePath)
	if err != nil {
		return err
	}

	fmt.Fprintf(docsFile, `# %s

{%% hint style="info" %%}
%s
{%% endhint %%}`, label, operation[0].docsHint)

	// sort for stability
	sort.Slice(operation, func(i, j int) bool {
		return operation[i].pathURL+operation[i].method > operation[j].pathURL+operation[j].method
	})
	for _, op := range operation {
		relativePathToSpec, err := filepath.Rel(path.Dir(filePath), path.Join(docsPath, op.specPath))
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(docsFile,
			`
{%% swagger src="%s" path="%s" method="%s" %%}
[spec.yaml](%s)
{%% endswagger %%}
`,
			relativePathToSpec,
			op.pathURL,
			strings.ToLower(op.method),
			relativePathToSpec,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func labelToFileName(label string) string {
	replacements := []string{"(", ")"}
	for _, replacement := range replacements {
		label = strings.ReplaceAll(label, replacement, "")
	}
	return strings.ToLower(strings.ReplaceAll(label, " ", "-")) + ".md"
}