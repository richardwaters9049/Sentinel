# Sentinel brand

Sentinel uses the detection-lens identity: cyan and violet signal ribbons frame a faceted core, with orbital details representing connected observations.

The short description is **Threat hunting & security analytics**. The existing application palette remains in use; the brand does not change finding severity colours.

## Application assets

- `apps/console/public/brand/sentinel-logo.svg`: detailed vector mark for sign-in and navigation.
- `apps/console/public/brand/sentinel-symbol.svg`: compact mark for workspace headers.
- `apps/console/components/sentinel/brand.tsx`: shared accessible wordmark and description.
- `apps/console/app/icon.svg`, `favicon.ico` and `apple-icon.png`: application icons.

Keep the SVG backgrounds transparent and retain the viewBox. Use the shared component rather than copying a lockup into each page. Decorative marks have empty alt text; the home link has a single accessible name. Persistent navigation branding remains static.

The editable brand board is [available in Figma](https://www.figma.com/design/IVpm49WFkC0Nb5uBxB6Cal?node-id=3-2). The application uses a vector reconstruction of the agreed detection-lens direction; the later raster preview is a visual reference, not a bundled screenshot.

## Opening sequence

The console opens with a brief signal-convergence animation: cyan/violet light trails,
a tracing orbit, the approved lens mark and a staggered wordmark. The tagline is
**Find the signal beneath the noise.** This is a brand introduction, not a service-health
or loading-progress indicator.

It plays once per browser tab (sessionStorage key `sentinel:opening:v1`), takes roughly
three seconds including the fade, and is skipped for reduced-motion preferences. The
Skip intro button and Escape dismiss it immediately. Focus returns to the application
after dismissal; underlying controls are inert while the intro is visible. Server
rendering leaves the application usable before hydration or without JavaScript.

To preview again in the same tab, clear that sessionStorage key and reload, or open a
fresh tab. Storage denial does not block the application.
