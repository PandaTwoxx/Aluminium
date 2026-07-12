package graph

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/PandaTwoxx/Aluminium/internal/client"
	"github.com/PandaTwoxx/Aluminium/internal/config"
)

type Node struct {
	Name         string
	Version      string
	BuildSystem  string
	Dependencies []string
	BuildSetup   *client.BuildSetup
	ServerURL    string
}

type DependencyGraph map[string]*Node

// ParseSpec splits a package spec like "libpng@1.6.37" into name and version.
func ParseSpec(spec string) (string, string) {
	parts := strings.SplitN(spec, "@", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], ""
}

// CompareVersions compares two version strings (semver-like) and returns:
// -1 if v1 < v2
// 1 if v1 > v2
// 0 if v1 == v2
func CompareVersions(v1, v2 string) int {
	p1 := strings.SplitN(v1, "-", 2)
	p2 := strings.SplitN(v2, "-", 2)

	base1, pre1 := p1[0], ""
	if len(p1) > 1 {
		pre1 = p1[1]
	}

	base2, pre2 := p2[0], ""
	if len(p2) > 1 {
		pre2 = p2[1]
	}

	parts1 := strings.Split(base1, ".")
	parts2 := strings.Split(base2, ".")
	for i := 0; i < len(parts1) && i < len(parts2); i++ {
		n1, err1 := strconv.Atoi(parts1[i])
		n2, err2 := strconv.Atoi(parts2[i])
		if err1 == nil && err2 == nil {
			if n1 != n2 {
				if n1 < n2 {
					return -1
				}
				return 1
			}
		} else {
			if parts1[i] != parts2[i] {
				if parts1[i] < parts2[i] {
					return -1
				}
				return 1
			}
		}
	}
	if len(parts1) < len(parts2) {
		return -1
	}
	if len(parts1) > len(parts2) {
		return 1
	}

	if pre1 == "" && pre2 != "" {
		return 1
	}
	if pre1 != "" && pre2 == "" {
		return -1
	}
	if pre1 != "" && pre2 != "" {
		if pre1 < pre2 {
			return -1
		}
		if pre1 > pre2 {
			return 1
		}
	}
	return 0
}

// ResolveGraph recursively searches searchServers in order for targets and build the graph.
func ResolveGraph(targets []string, cfg *config.Config, api *client.APIClient) (DependencyGraph, error) {
	g := make(DependencyGraph)

	var resolve func(spec string) error
	resolve = func(spec string) error {
		name, version := ParseSpec(spec)
		if _, exists := g[name]; exists {
			return nil
		}

		var foundPkg *client.Package
		var foundServer string

		for _, server := range cfg.SearchServers {
			token := cfg.Servers[server].Token
			if version != "" {
				pkg, err := api.GetPackage(server, name, version, token)
				if err == nil && pkg != nil {
					foundPkg = pkg
					foundServer = server
					break
				}
			} else {
				pkgs, err := api.ListPackages(server, token)
				if err == nil {
					var latest *client.Package
					for _, p := range pkgs {
						if p.Name == name {
							if latest == nil || CompareVersions(p.Version, latest.Version) > 0 {
								pCopy := p
								latest = &pCopy
							}
						}
					}
					if latest != nil {
						foundPkg = latest
						foundServer = server
						break
					}
				}
			}
		}

		if foundPkg == nil {
			if version != "" {
				return fmt.Errorf("package %s@%s not found on any configured server", name, version)
			}
			return fmt.Errorf("package %s not found on any configured server", name)
		}

		g[name] = &Node{
			Name:         foundPkg.Name,
			Version:      foundPkg.Version,
			BuildSystem:  foundPkg.BuildSystem,
			Dependencies: foundPkg.Dependencies,
			BuildSetup:   foundPkg.BuildSetup,
			ServerURL:    foundServer,
		}

		for _, depSpec := range foundPkg.Dependencies {
			if err := resolve(depSpec); err != nil {
				return err
			}
		}

		return nil
	}

	for _, t := range targets {
		if err := resolve(t); err != nil {
			return nil, err
		}
	}

	return g, nil
}

// TopoSort performs a topological sort on the graph and returns the sorted package names.
// It returns an error if a cycle is detected.
func TopoSort(graph DependencyGraph, targets []string) ([]string, error) {
	var result []string
	state := make(map[string]int) // 0: unvisited, 1: visiting, 2: visited
	var path []string

	var visit func(name string) error
	visit = func(name string) error {
		if state[name] == 1 {
			cycleStartIdx := -1
			for i, p := range path {
				if p == name {
					cycleStartIdx = i
					break
				}
			}
			var cyclePath []string
			if cycleStartIdx != -1 {
				cyclePath = append(path[cycleStartIdx:], name)
			} else {
				cyclePath = append(path, name)
			}
			return fmt.Errorf("dependency cycle detected: %s", strings.Join(cyclePath, " -> "))
		}
		if state[name] == 2 {
			return nil
		}

		state[name] = 1
		path = append(path, name)

		node, exists := graph[name]
		if !exists {
			return fmt.Errorf("missing dependency in graph: %s", name)
		}

		for _, depSpec := range node.Dependencies {
			depName, _ := ParseSpec(depSpec)
			if err := visit(depName); err != nil {
				return err
			}
		}

		state[name] = 2
		path = path[:len(path)-1]
		result = append(result, name)
		return nil
	}

	for _, t := range targets {
		name, _ := ParseSpec(t)
		if err := visit(name); err != nil {
			return nil, err
		}
	}

	return result, nil
}
