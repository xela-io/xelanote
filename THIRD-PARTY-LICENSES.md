# Third-Party Licenses

This project includes third-party software components under various open-source licenses.

## LGPL-3.0-or-later

### sharp / libvips

The [`sharp`](https://sharp.pixelplumbing.com/) image processing library (used as a devDependency for image optimization) includes native bindings from [libvips](https://www.libvips.org/) which are licensed under **LGPL-3.0-or-later**.

- **Packages:** `@img/sharp-libvips-linux-x64`, `@img/sharp-libvips-linuxmusl-x64`
- **Source:** <https://github.com/lovell/sharp-libvips>
- **License:** LGPL-3.0-or-later
- **Usage:** Dynamically linked native bindings (not statically compiled into the application)
- **Full license text:** <https://www.gnu.org/licenses/lgpl-3.0.html>

## OFL-1.1 (SIL Open Font License)

The following fonts are bundled via Fontsource:

- **DM Sans** -- [Google Fonts](https://fonts.google.com/specimen/DM+Sans) (OFL-1.1)
- **JetBrains Mono** -- [JetBrains](https://www.jetbrains.com/lp/mono/) (OFL-1.1)
- **Literata** -- [Google Fonts](https://fonts.google.com/specimen/Literata) (OFL-1.1)

## ISC

- **Lucide Icons** -- [lucide.dev](https://lucide.dev/) (ISC)

## All Other Dependencies

All remaining dependencies use permissive licenses (MIT, Apache-2.0, BSD-2-Clause, BSD-3-Clause, ISC). See `backend/go.mod` and `frontend/package.json` for the full dependency list.
