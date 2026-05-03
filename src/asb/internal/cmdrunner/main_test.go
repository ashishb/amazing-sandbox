package cmdrunner

import "flag"

// testMode is the sandbox execution mode passed to tests (e.g. -mode=native).
var testMode = flag.String("mode", "docker", "sandbox mode to use (docker or native)")
