package dockerfile

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

const dockerfileName = "Dockerfile"

var (
	rxFrom  = regexp.MustCompile(`^FROM (?P<ref>(?P<image>[^:@\s]+):?(?P<tag>[^\s@]+)?@?(?P<digest>sha256:.*)?)(?: as .*)?$`) //nolint:lll
	rxLabel = regexp.MustCompile(`^LABEL (\S+)="(\S+)"$`)
)

// Dockerfile holds the information from a Dockerfile.
type Dockerfile struct {
	ContextPath string
	Filename    string
	From        []ImageRef
	Labels      map[string]string
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

// IsDockerfile checks whether a file is a Dockerfile.
func IsDockerfile(filename string) bool {
	return strings.HasSuffix(filename, dockerfileName)
}

// ParseDockerfile parses an actual Dockerfile, and creates an instance of a Dockerfile struct.
func ParseDockerfile(filename string) (*Dockerfile, error) {
	logrus.Debugf("Parsing dockerfile \"%s\"", filename)
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var dckFile Dockerfile
	dckFile.ContextPath = path.Dir(filename)
	dckFile.Filename = path.Base(filename)
	dckFile.Labels = map[string]string{}

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
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	logrus.Debugf("Successfully parsed dockerfile. From=%v, Labels=%v", dckFile.From, dckFile.Labels)

	return &dckFile, nil
}

// ReplaceTags replaces all matching tags by a replacement tag.
// The diff map keys are source image refs, and their values are the replacement refs.
// Many references to images may be replaced in the Dockerfile,
// those from the FROM statements, and also --from arguments.
func ReplaceTags(d Dockerfile, diff map[string]string) error {
	for ref, newRef := range diff {
		err := replace(path.Join(d.ContextPath, d.Filename), ref, newRef)
		if err != nil {
			return fmt.Errorf("cannot replace \"%s\" with \"%s\": %w", ref, newRef, err)
		}
	}
	return nil
}

// ResetTags resets the tags that were replaced by ReplaceTags by doing the opposite process.
// The diff map is the same map that was passed previously to ReplaceTags.
// Again, many references to images may be replaced in the Dockerfile,
// those from the FROM statements, and also --from arguments.
func ResetTags(d Dockerfile, diff map[string]string) error {
	for initialRef, newRef := range diff {
		err := replace(path.Join(d.ContextPath, d.Filename), newRef, initialRef)
		if err != nil {
			return fmt.Errorf("cannot reset tag \"%s\" to \"%s\": %w", newRef, initialRef, err)
		}
	}
	return nil
}

func replace(path string, previous string, next string) error {
	read, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	newContents := strings.ReplaceAll(string(read), previous, next)
	if err = os.WriteFile(path, []byte(newContents), 0); err != nil {
		return err
	}
	return nil
}
