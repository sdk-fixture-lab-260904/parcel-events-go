// The SINGLE Go-side source of the Go kernel version (doc 38 §1: `sdkSha`
// includes `kernelVersion`; §3.4: a kernel change is a version input, never
// invisible drift). Bump on ANY behavioral change to a vendored kernel file.
// The TS-side `version.ts` mirrors these constants; the `go-kernel-manifest`
// determinism gate pins the two byte-for-byte, so drift is a red build.
//
// Vendored kernel file — imports nothing (self-contained by construction).

package kernel

// KernelVersion is the semver of the hand-written Go runtime kernel.
const KernelVersion = "0.1.0"

// KernelName is the kernel's User-Agent product token.
const KernelName = "doctorine-go-kernel"

// UserAgent builds the User-Agent value stamped on every request:
// "<sdk>/<version> doctorine-go-kernel/<kernelVersion>". The generated
// client passes its sdkSha-stamped version, so every request carries the
// exact build identity.
func UserAgent(sdkName, sdkVersion string) string {
	return sdkName + "/" + sdkVersion + " " + KernelName + "/" + KernelVersion
}
