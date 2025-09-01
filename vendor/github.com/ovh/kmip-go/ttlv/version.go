package ttlv

import (
	"cmp"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Version interface is implemented by types representing
// a protocol version when the protocol supports versioning like KMIP.
type Version interface {
	// Major returns the version major number
	Major() int
	// Minor returns the version minor number
	Minor() int
}

// CompareVersions compare 2 protocol version and returns
//
//	-1 if a is less than b,
//	 0 if a equals b,
//	+1 if a is greater than b.
func CompareVersions[A, B Version](a A, b B) int {
	if c := cmp.Compare(a.Major(), b.Major()); c != 0 {
		return c
	}
	return cmp.Compare(a.Minor(), b.Minor())
}

type extension struct {
	version *version
}

// versionIn checks that the version stored in the extension is in the given
// version range `vrange`. If extension has no version set, then the function always
// returns true.
func (ext *extension) versionIn(vrange versionRange) bool {
	if ext.version == nil {
		return true
	}
	// fmt.Printf("Check if %s is in %s\n", *ext.version, vrange)
	return ext.version.isIn(vrange)
}

func (ext *extension) setVersion(v Version) {
	ext.version = &version{major: v.Major(), minor: v.Minor()}
	// fmt.Println("set encoder version to", **enc.version)
}

type version struct {
	major int
	minor int
}

func (v version) String() string {
	return fmt.Sprintf("%d.%d", v.major, v.minor)
}

func (v version) Major() int {
	return v.major
}

func (v version) Minor() int {
	return v.minor
}

func parseVersion(s string) (version, error) {
	var v version
	var err error
	major, minor, found := strings.Cut(s, ".")
	if !found {
		return v, errors.New("cannot parse protocol version string")
	}
	major = strings.TrimPrefix(major, "v")
	v.major, err = strconv.Atoi(major)
	if err != nil {
		return v, err
	}
	v.minor, err = strconv.Atoi(minor)
	if err != nil {
		return v, err
	}
	return v, nil
}

func (v version) compare(other Version) int {
	return CompareVersions(v, other)
}

func (v version) isIn(rng versionRange) bool {
	return rng.contains(v)
}

type versionRange struct {
	start *version
	end   *version
}

func parseVersionRange(s string) (versionRange, error) {
	start, end, found := strings.Cut(s, "..")
	if !found {
		v, err := parseVersion(s)
		if err != nil {
			return versionRange{}, err
		}
		return versionRange{start: &v, end: &v}, nil
	}

	var vrange = versionRange{}
	if start != "" {
		st, err := parseVersion(start)
		if err != nil {
			return vrange, err
		}
		vrange.start = &st
	}
	if end != "" {
		en, err := parseVersion(end)
		if err != nil {
			return vrange, err
		}
		vrange.end = &en
	}
	if vrange.start != nil && vrange.end != nil && CompareVersions(vrange.start, vrange.end) > 0 {
		return versionRange{}, errors.New("Invalid range: start is greater than end")
	}
	return vrange, nil
}

func (rng versionRange) contains(v Version) bool {
	if rng.start != nil && rng.start.compare(v) > 0 {
		return false
	}
	if rng.end != nil && rng.end.compare(v) < 0 {
		return false
	}
	return true
}

func (rng versionRange) String() string {
	start := ""
	end := ""
	if rng.start != nil {
		start = rng.start.String()
	}
	if rng.end != nil {
		end = rng.end.String()
	}
	return fmt.Sprintf("%s..%s", start, end)
}
