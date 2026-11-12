package cmdrunner

// _bwrapSystemDir is a host directory that is exposed read-only inside the bubblewrap sandbox.
type _bwrapSystemDir struct {
	path string
	// Optional directories don't exist on every distribution, e.g. /snap only exists when snapd is
	// installed and /lib64 does not exist on Debian/Ubuntu arm64. bwrap aborts with
	// "Can't find source path" when the source of a --ro-bind does not exist, so these use
	// --ro-bind-try instead.
	optional bool
}

var _bwrapSystemDirs = []_bwrapSystemDir{
	{path: "/bin", optional: true},
	{path: "/etc", optional: false},
	{path: "/lib", optional: true},
	{path: "/lib64", optional: true},
	{path: "/sbin", optional: true},
	{path: "/opt", optional: true},
	{path: "/run", optional: true},
	{path: "/snap", optional: true},
	{path: "/sys", optional: true},
	{path: "/usr", optional: false},
}

// getBwrapSystemDirMountArgs returns the bwrap arguments that expose the system directories read-only
func getBwrapSystemDirMountArgs() []string {
	args := make([]string, 0, 3*len(_bwrapSystemDirs))
	for _, dir := range _bwrapSystemDirs {
		bindOption := "--ro-bind"
		if dir.optional {
			bindOption = "--ro-bind-try"
		}
		args = append(args, bindOption, dir.path, dir.path)
	}
	return args
}
