package gpupaas

// Version is the SDK release this source carries: the git tag without the leading v.
const Version = "0.1.0"

// ControllerAPI names the controller API line this SDK line speaks to, written as "v<MAJOR>.<MINOR>" of Version.
// Every tag vX.Y.Z of one line talks to the same controller line; the README table "Controller compatibility"
// maps each line to the controller releases it was verified against. A controller generation that changes the
// API gets its own line (a release/vX.Y branch), never a patch on another one.
const ControllerAPI = "v0.1"
