package dockerfile

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/radiofrance/dib/pkg/logger"
)

const dockerfileName = "Dockerfile"

var (
	rxFrom  = regexp.MustCompile(`^FROM (?P<ref>(?P<image>[^:@\s]+):?(?P<tag>[^\s@]+)?@?(?P<digest>sha256:.*)?)(?: as .*)?$`) //nolint:lll
	rxLabel = regexp.MustCompile(`^LABEL (\S+)="(\S+)"$`)
	rxArg   = regexp.MustCompile(`^ARG\s+([a-zA-Z_]\w*)(\s*=\s*[^#\n]*)?$`)
)

// Dockerfile holds the information from a Dockerfile.
type Dockerfile struct {
	ContextPath string
	Filename    string
	From        []ImageRef
	Labels      map[string]string
	Args        map[string]string
}

// ImageRef holds the information about an image reference present in FROM statements.
type ImageRef struct {
	Name   string
	Tag    string
	Digest string
}

func (d *Dockerfile) addFrom(from ImageRef) {
	d.From = append(d.From, from)
}

func (d *Dockerfile) addLabel(name string, value string) {
	d.Labels[name] = value
}

func (d *Dockerfile) addArg(name string, value string) {
	d.Args[name] = value
}

// IsDockerfile checks whether a file is a Dockerfile.
func IsDockerfile(filename string) bool {
	return strings.HasSuffix(filename, dockerfileName)
}

// ParseDockerfile parses an actual Dockerfile, and creates an instance of a Dockerfile struct.
func ParseDockerfile(filename string) (*Dockerfile, error) {
	logger.Debugf("Parsing dockerfile \"%s\"", filename)

	file, err := os.Open(filename) //nolint:gosec
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = file.Close()
	}()

	var dckFile Dockerfile

	dckFile.ContextPath = path.Dir(filename)
	dckFile.Filename = path.Base(filename)
	dckFile.Labels = map[string]string{}
	dckFile.Args = map[string]string{}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		txt := scanner.Text()

		switch {
		case rxFrom.MatchString(txt):
			match := rxFrom.FindStringSubmatch(txt)
			result := make(map[string]string)

			for i, name := range rxFrom.SubexpNames() {
				if i != 0 && name != "" {
					result[name] = match[i]
				}
			}

			dckFile.addFrom(ImageRef{
				Name:   result["image"],
				Tag:    result["tag"],
				Digest: result["digest"],
			})
		case rxLabel.MatchString(txt):
			result := rxLabel.FindStringSubmatch(txt)
			dckFile.addLabel(result[1], result[2])
		case rxArg.MatchString(txt):
			result := rxArg.FindStringSubmatch(txt)
			dckFile.addArg(result[1], txt)
		}
	}

	err = scanner.Err()
	if err != nil {
		return nil, err
	}

	logger.Debugf("Successfully parsed dockerfile. From=%v, Labels=%v, Args=%v",
		dckFile.From, dckFile.Labels, dckFile.Args)

	return &dckFile, nil
}

// ReplaceInFile replaces all matching references by a replacement.
// The diff map keys are source references, and the values are replacements.
// Many references to images may be replaced, those from the FROM statements, and also --from arguments.
func ReplaceInFile(path string, diff map[string]string) error {
	for ref, newRef := range diff {
		err := replace(path, ref, newRef)
		if err != nil {
			return fmt.Errorf("cannot replace \"%s\" with \"%s\": %w", ref, newRef, err)
		}
	}

	return nil
}

// ResetFile resets the strings that were replaced by ReplaceInFile by doing the opposite process.
func ResetFile(path string, diff map[string]string) error {
	for initialRef, newRef := range diff {
		err := replace(path, newRef, initialRef)
		if err != nil {
			return fmt.Errorf("cannot reset tag \"%s\" to \"%s\": %w", newRef, initialRef, err)
		}
	}

	return nil
}

func replace(path string, previous string, next string) error {
	read, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		return err
	}

	newContents := strings.ReplaceAll(string(read), previous, next)

	return os.WriteFile(path, []byte(newContents), 0)
}
