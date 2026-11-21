# Packaging

`frontend/images/logo.svg` is the source logo. The checked-in `frontend/images/logo.png` is the raster asset used
by Windows executable resources and MSIX package icons. Windows builds also generate an ignored
`rsrc_windows_amd64.syso` file from `frontend/images/logo.png`. The resource embeds the AI Gauge icon and Windows file metadata into `ai-gauge.exe`. Install the
resource generator once with `go install github.com/tc-hib/go-winres@v0.3.3` if it is not already
available.
The MSIX manifest supplies the Store icons on its own, so the PR-check workflows (`msix.yml`,
`pull-request.yml`) skip this step for speed. The release workflow (`release.yml`) does not skip
it: `ai-gauge.exe` is also uploaded to GitHub Releases as the portable executable, and without the
embedded resource that file has no icon at all in Explorer/the taskbar.

`.\build.ps1 checks` also creates a local MSIX and verifies that its package name and staged
manifest version match the requested version (local builds default to `0.0.0`). Local packages use
an explicit suffix such as `dist/ai-gauge_0.2.4.0_x64_local.msix`; the release workflow alone
produces the canonical `ai-gauge_0.2.4.0_x64.msix` asset. The versioned local package remains in
`dist/` for inspection. Remove generated packaging output explicitly when it is no longer needed:

```powershell
.\build.ps1 clean
```

Build a local MSIX package directly with:

```powershell
.\build.ps1 package
```

Release tags are the application version source of truth. For example, `v0.2.1` becomes the four-part
MSIX version `0.2.1.0`; pass the same tag to `build.ps1 -Version` for a matching local package. The staging directory is `dist/staging/`; the generated package is written
to `dist/` and ignored by Git.

The packaging script locates `makeappx.exe` from the Windows SDK. If it is not on `PATH`, pass its
full path through the existing packaging script parameter. `signtool.exe` is only needed when
creating a locally signed package.

## Releases

Releases are triggered by pushing a stable `vX.Y.Z` tag:

```powershell
git tag v0.6.2
git push origin v0.6.2
```

The `Release` GitHub Actions workflow (`.github/workflows/release.yml`) builds the MSIX and the
standalone portable executable (`ai-gauge_<version>_x64.exe`). It attaches the executable and its
`SHA256SUMS.txt` to the GitHub release, creating the release or, if it already exists (for example
created from the GitHub UI), uploading to it. It publishes the MSIX to the Microsoft Store, which
signs it; the unsigned MSIX is not attached to the GitHub release because it cannot be installed as is.

Once the `jmnote.ai-gauge` package exists in [winget-pkgs](https://github.com/microsoft/winget-pkgs),
the workflow also submits each release to winget with `wingetcreate update`, pointing at the portable
executable. This needs a `WINGET_TOKEN` secret: a classic personal access token with the
`public_repo` scope, which `wingetcreate` uses to push to a fork of winget-pkgs and open the pull
request. The first version of the package is submitted by hand.

The portable `.exe` runs unsigned and needs no installation - unlike the MSIX, which either goes through
Store certification or needs a certificate matching the package Publisher installed and trusted
first. Running the portable `.exe` still triggers SmartScreen on a machine that has not seen it
before; that is a separate, much smaller prompt than installing a certificate.

> For Microsoft Store publishing, Partner Center credentials, and submission details, see
> [`hack/msstore/msstore.md`](../hack/msstore/msstore.md).
