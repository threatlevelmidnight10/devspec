package spec

import (
	"fmt"
	"strconv"
	"strings"
)

type yamlLine struct {
	indent int
	text   string
	lineNo int
}

func parseYAML(input []byte) (map[string]any, error) {
	lines, err := lexYAMLLines(string(input))
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty document")
	}

	idx := 0
	root, err := parseMap(lines, &idx, 0)
	if err != nil {
		return nil, err
	}
	if idx != len(lines) {
		return nil, fmt.Errorf("unexpected trailing content at line %d", lines[idx].lineNo)
	}
	return root, nil
}

func lexYAMLLines(src string) ([]yamlLine, error) {
	raw := strings.Split(src, "\n")
	out := make([]yamlLine, 0, len(raw))
	for i, line := range raw {
		trimmedRight := strings.TrimRight(line, " \t\r")
		if strings.TrimSpace(trimmedRight) == "" {
			continue
		}
		l := strings.TrimLeft(trimmedRight, " ")
		if strings.HasPrefix(l, "#") {
			continue
		}
		indent := len(trimmedRight) - len(l)
		if indent%2 != 0 {
			return nil, fmt.Errorf("line %d: indentation must be multiples of 2 spaces", i+1)
		}
		out = append(out, yamlLine{indent: indent, text: l, lineNo: i + 1})
	}
	return out, nil
}

func parseMap(lines []yamlLine, idx *int, indent int) (map[string]any, error) {
	result := map[string]any{}
	for *idx < len(lines) {
		line := lines[*idx]
		if line.indent < indent {
			break
		}
		if line.indent > indent {
			return nil, fmt.Errorf("line %d: unexpected indentation", line.lineNo)
		}
		if strings.HasPrefix(line.text, "- ") {
			return nil, fmt.Errorf("line %d: list item where map key expected", line.lineNo)
		}

		key, value, hasInline, err := parseKeyValue(line)
		if err != nil {
			return nil, err
		}
		*idx += 1

		if hasInline {
			result[key] = parseScalar(value)
			continue
		}

		if *idx >= len(lines) || lines[*idx].indent <= indent {
			result[key] = map[string]any{}
			continue
		}

		next := lines[*idx]
		if next.indent != indent+2 {
			return nil, fmt.Errorf("line %d: child indentation must be %d", next.lineNo, indent+2)
		}
		if strings.HasPrefix(next.text, "- ") {
			v, err := parseList(lines, idx, indent+2)
			if err != nil {
				return nil, err
			}
			result[key] = v
			continue
		}
		v, err := parseMap(lines, idx, indent+2)
		if err != nil {
			return nil, err
		}
		result[key] = v
	}
	return result, nil
}

func parseList(lines []yamlLine, idx *int, indent int) ([]any, error) {
	result := []any{}
	for *idx < len(lines) {
		line := lines[*idx]
		if line.indent < indent {
			break
		}
		if line.indent > indent {
			return nil, fmt.Errorf("line %d: unexpected indentation in list", line.lineNo)
		}
		if !strings.HasPrefix(line.text, "- ") {
			break
		}
		item := strings.TrimSpace(strings.TrimPrefix(line.text, "- "))
		*idx += 1

		if item == "" {
			if *idx >= len(lines) || lines[*idx].indent <= indent {
				result = append(result, "")
				continue
			}
			next := lines[*idx]
			if next.indent != indent+2 {
				return nil, fmt.Errorf("line %d: child indentation must be %d", next.lineNo, indent+2)
			}
			if strings.HasPrefix(next.text, "- ") {
				child, err := parseList(lines, idx, indent+2)
				if err != nil {
					return nil, err
				}
				result = append(result, child)
				continue
			}
			child, err := parseMap(lines, idx, indent+2)
			if err != nil {
				return nil, err
			}
			result = append(result, child)
			continue
		}

		if key, value, hasInline, err := parseInlineMap(item); err == nil && hasInline {
			result = append(result, map[string]any{key: parseScalar(value)})
			continue
		}
		result = append(result, parseScalar(item))
	}
	return result, nil
}

func parseKeyValue(line yamlLine) (key, value string, hasInline bool, err error) {
	parts := strings.SplitN(line.text, ":", 2)
	if len(parts) != 2 {
		return "", "", false, fmt.Errorf("line %d: expected key: value", line.lineNo)
	}
	key = strings.TrimSpace(parts[0])
	if key == "" {
		return "", "", false, fmt.Errorf("line %d: empty key", line.lineNo)
	}
	value = strings.TrimSpace(parts[1])
	return key, value, value != "", nil
}

func parseInlineMap(item string) (key, value string, hasInline bool, err error) {
	parts := strings.SplitN(item, ":", 2)
	if len(parts) != 2 {
		return "", "", false, nil
	}
	key = strings.TrimSpace(parts[0])
	value = strings.TrimSpace(parts[1])
	if key == "" {
		return "", "", false, fmt.Errorf("inline map has empty key")
	}
	if value == "" {
		return "", "", false, nil
	}
	return key, value, true, nil
}

func parseScalar(raw string) any {
	if raw == "true" {
		return true
	}
	if raw == "false" {
		return false
	}
	if i, err := strconv.Atoi(raw); err == nil {
		return i
	}
	if strings.HasPrefix(raw, `"`) && strings.HasSuffix(raw, `"`) && len(raw) >= 2 {
		return strings.Trim(raw, `"`)
	}
	if strings.HasPrefix(raw, `'`) && strings.HasSuffix(raw, `'`) && len(raw) >= 2 {
		return strings.Trim(raw, `'`)
	}
	return raw
}
