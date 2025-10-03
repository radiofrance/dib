package strutil

import "strings"

// ConvertKVStringsToMap is from https://github.com/moby/moby/blob/v20.10.0-rc2/runconfig/opts/parse.go
//
// ConvertKVStringsToMap converts ["key=value"] to {"key":"value"}.
func ConvertKVStringsToMap(values []string) map[string]string {
	result := make(map[string]string, len(values))

	const splitLimit = 2
	for _, value := range values {
		kv := strings.SplitN(value, "=", splitLimit)
		if len(kv) == 1 {
			result[kv[0]] = ""
		} else {
			result[kv[0]] = kv[1]
		}
	}

	return result
}

func DedupeStrSlice(in []string) []string {
	m := make(map[string]struct{})

	var res []string

	for _, s := range in {
		if _, ok := m[s]; !ok {
			res = append(res, s)
			m[s] = struct{}{}
		}
	}

	return res
}
