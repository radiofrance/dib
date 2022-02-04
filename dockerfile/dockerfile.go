package dockerfile

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	dockerfileName = "Dockerfile"
	dibPlaceholder = "DIB_MANAGED_VERSION"
)

var (
	rxFrom  *regexp.Regexp
	rxLabel *regexp.Regexp
)

func init() {
	rxFrom = regexp.MustCompile(`^FROM (\S+):\S+( as \S+)?$`)
	rxLabel = regexp.MustCompile(`^LABEL (\S+)="(\S+)"$`)
}

// Dockerfile holds the information from a Dockerfile.
type Dockerfile struct {
	ContextPath string
	Filename    string
	From        []string
	Labels      map[string]string
}

func (d *Dockerfile) addFrom(from string) {
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
			dckFile.addFrom(rxFrom.FindStringSubmatch(txt)[1])
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
// The diff map keys are source images, and their values are the replacement tags.
// Many references to images may be replaced in the Dockerfile,
// those from the FROM statements, and also --from arguments.
func ReplaceTags(d Dockerfile, diff map[string]string) error {
	for image, newTag := range diff {
		match := fmt.Sprintf("%s:%s", image, dibPlaceholder)
		err := replace(path.Join(d.ContextPath, d.Filename), match, newTag)
		if err != nil {
			return fmt.Errorf("cannot replace \"%s\" with \"%s\": %w", match, newTag, err)
		}
	}
	return nil
}

// ResetTags resets the tags that were replaced by ReplaceTags by doing the opposite process.
// The diff map is the same map that was passed previously to ReplaceTags.
// Again, many references to images may be replaced in the Dockerfile,
// those from the FROM statements, and also --from arguments.
func ResetTags(d Dockerfile, diff map[string]string) error {
	for image, newTag := range diff {
		initialValue := fmt.Sprintf("%s:%s", image, dibPlaceholder)
		err := replace(path.Join(d.ContextPath, d.Filename), newTag, initialValue)
		if err != nil {
			return fmt.Errorf("cannot reset tag \"%s\" to \"%s\": %w", newTag, initialValue, err)
		}
	}
	return nil
}

func replace(path string, previous string, next string) error {
	read, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	newContents := strings.ReplaceAll(string(read), previous, next)
	if err = ioutil.WriteFile(path, []byte(newContents), 0); err != nil {
		return err
	}
	return nil
}
