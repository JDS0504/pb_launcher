import {
  ArrowRight,
  BadgeCheck,
  Code2,
  Download,
  Github,
  Globe2,
  LockKeyhole,
  Rocket,
  ServerCog,
  ShieldCheck,
  Workflow,
} from "lucide-react";
import { Link } from "react-router";

const GITHUB_URL = "https://github.com/user0608/pb_launcher";
const RELEASES_URL = `${GITHUB_URL}/releases/latest`;

const features = [
  {
    icon: ServerCog,
    title: "Manage many PocketBase instances",
    description:
      "Create, start, stop, restart, clone, and upgrade PocketBase instances from a clean web dashboard.",
  },
  {
    icon: Globe2,
    title: "Built-in domains and proxy",
    description:
      "Each instance can get its own URL, custom domains, and routing through the integrated reverse proxy.",
  },
  {
    icon: LockKeyhole,
    title: "HTTPS certificate automation",
    description:
      "Use self-signed certificates for local setups or ACME providers such as Cloudflare for production domains.",
  },
  {
    icon: Download,
    title: "PocketBase release sync",
    description:
      "Track PocketBase releases from configured repositories and download binaries only when needed.",
  },
  {
    icon: ShieldCheck,
    title: "Backups, clones, and snapshots",
    description:
      "Create local ZIP backups, restore instances, clone stopped services, and keep point-in-time snapshots.",
  },
  {
    icon: Code2,
    title: "Single Go binary",
    description:
      "PBLauncher embeds the UI and runs as a lightweight Go service with PocketBase as its backing database.",
  },
];

const useCases = [
  "Self-host PocketBase apps on a VPS",
  "Run isolated PocketBase environments for clients or projects",
  "Create disposable staging and development instances",
  "Manage domains, SSL, backups, and upgrades from one place",
];

export const LandingPage = () => {
  return (
    <div className="min-h-screen bg-background text-foreground font-sans">
      <header className="sticky top-0 z-30 border-b border-border/70 bg-background/85 backdrop-blur">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-4 sm:px-6 lg:px-8">
          <Link to="/" className="flex items-center gap-3" aria-label="PBLauncher home">
            <img src="/logo_circle.png" alt="" className="h-10 w-10" />
            <span className="text-xl font-bold tracking-tight">PBLauncher</span>
          </Link>
          <nav className="hidden items-center gap-6 text-sm text-subtle md:flex">
            <a href="#features" className="hover:text-foreground">
              Features
            </a>
            <a href="#use-cases" className="hover:text-foreground">
              Use cases
            </a>
            <Link to="/docs" className="hover:text-foreground">
              Docs
            </Link>
            <a
              href={GITHUB_URL}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 rounded-full border border-border px-4 py-2 text-foreground hover:border-accent"
            >
              <Github className="h-4 w-4" />
              GitHub
            </a>
          </nav>
        </div>
      </header>

      <main>
        <section className="relative overflow-hidden border-b border-border/60">
          <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(255,120,73,0.24),transparent_34%),radial-gradient(circle_at_bottom_right,rgba(92,124,250,0.16),transparent_30%)]" />
          <div className="relative mx-auto grid max-w-[90rem] gap-12 px-4 py-16 sm:px-6 md:py-20 lg:grid-cols-[0.85fr_1.15fr] lg:items-center lg:px-8 lg:py-28 xl:grid-cols-[0.8fr_1.2fr]">
            <div className="flex flex-col justify-center">
              <div className="mb-6 inline-flex w-fit items-center gap-2 rounded-full border border-accent/40 bg-accent/10 px-4 py-2 text-sm text-accent-light">
                <BadgeCheck className="h-4 w-4" />
                Open source PocketBase instance manager
              </div>
              <h1 className="max-w-4xl text-4xl font-black tracking-tight text-foreground sm:text-6xl lg:text-7xl">
                Host and manage PocketBase instances with one lightweight tool.
              </h1>
              <p className="mt-6 max-w-2xl text-lg leading-8 text-subtle sm:text-xl">
                PBLauncher is a Go-based, self-hosted control panel for creating,
                launching, securing, backing up, and upgrading PocketBase
                instances from a single web UI.
              </p>
              <div className="mt-8 flex flex-col gap-3 sm:flex-row">
                <a
                  href={RELEASES_URL}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center justify-center gap-2 rounded-xl bg-accent px-6 py-3 font-semibold text-white shadow-lg shadow-accent/20 transition hover:bg-accent-hover"
                >
                  <Download className="h-5 w-5" />
                  Download latest release
                </a>
                <a
                  href={GITHUB_URL}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center justify-center gap-2 rounded-xl border border-border bg-surface/70 px-6 py-3 font-semibold text-foreground transition hover:border-accent"
                >
                  <Github className="h-5 w-5" />
                  View source code
                </a>
              </div>
              <dl className="mt-10 grid max-w-xl grid-cols-3 gap-4 text-sm">
                <div className="rounded-2xl border border-border bg-surface/70 p-4">
                  <dt className="text-muted">Runtime</dt>
                  <dd className="mt-1 font-semibold">Single binary</dd>
                </div>
                <div className="rounded-2xl border border-border bg-surface/70 p-4">
                  <dt className="text-muted">Backend</dt>
                  <dd className="mt-1 font-semibold">Go + PocketBase</dd>
                </div>
                <div className="rounded-2xl border border-border bg-surface/70 p-4">
                  <dt className="text-muted">License</dt>
                  <dd className="mt-1 font-semibold">Open source</dd>
                </div>
              </dl>
            </div>

            <div className="relative min-w-0 lg:-mr-6 xl:-mr-12">
              <div className="rounded-3xl border border-border bg-surface/80 p-2 shadow-2xl shadow-black/40 sm:p-3">
                <img
                  src="/screenshot.png"
                  alt="PBLauncher dashboard showing PocketBase instances and management actions"
                  className="h-auto w-full rounded-2xl border border-border object-cover"
                />
              </div>
              <div className="mt-3 rounded-2xl border border-border bg-background/95 p-4 shadow-xl backdrop-blur sm:absolute sm:-bottom-6 sm:left-6 sm:right-6 sm:mt-0 sm:p-5">
                <div className="flex items-start gap-3">
                  <Workflow className="mt-1 h-5 w-5 text-accent" />
                  <p className="text-sm leading-6 text-subtle">
                    Launch isolated PocketBase apps, route domains, manage SSL,
                    and keep operational tasks visible from one dashboard.
                  </p>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section id="features" className="mx-auto max-w-7xl px-4 py-20 sm:px-6 lg:px-8">
          <div className="max-w-3xl">
            <p className="text-sm font-semibold uppercase tracking-[0.2em] text-accent">
              Features
            </p>
            <h2 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl">
              Everything needed to operate PocketBase apps.
            </h2>
            <p className="mt-4 text-subtle">
              PBLauncher focuses on practical instance operations: provisioning,
              lifecycle control, domains, certificates, backups, and release
              management.
            </p>
          </div>
          <div className="mt-10 grid gap-5 md:grid-cols-2 lg:grid-cols-3">
            {features.map(feature => (
              <article
                key={feature.title}
                className="rounded-3xl border border-border bg-surface p-6 transition hover:-translate-y-1 hover:border-accent/60"
              >
                <feature.icon className="h-8 w-8 text-accent" />
                <h3 className="mt-5 text-xl font-semibold">{feature.title}</h3>
                <p className="mt-3 text-sm leading-6 text-subtle">
                  {feature.description}
                </p>
              </article>
            ))}
          </div>
        </section>

        <section id="use-cases" className="border-y border-border/60 bg-surface/45">
          <div className="mx-auto grid max-w-7xl gap-10 px-4 py-20 sm:px-6 lg:grid-cols-2 lg:px-8">
            <div>
              <p className="text-sm font-semibold uppercase tracking-[0.2em] text-accent">
                Use cases
              </p>
              <h2 className="mt-3 text-3xl font-bold tracking-tight sm:text-4xl">
                For developers who want PocketBase without repetitive server work.
              </h2>
              <p className="mt-5 text-subtle">
                PocketBase is simple. Running several PocketBase apps with
                domains, HTTPS, upgrades, and backups can become repetitive.
                PBLauncher packages that workflow into an open source dashboard.
              </p>
            </div>
            <div className="grid gap-4">
              {useCases.map(item => (
                <div
                  key={item}
                  className="flex items-center gap-3 rounded-2xl border border-border bg-background/70 p-4"
                >
                  <Rocket className="h-5 w-5 text-accent" />
                  <span className="font-medium">{item}</span>
                </div>
              ))}
            </div>
          </div>
        </section>

        <section className="mx-auto max-w-7xl px-4 py-20 sm:px-6 lg:px-8">
          <div className="rounded-3xl border border-border bg-gradient-to-br from-surface to-muted-dark p-8 sm:p-10 lg:flex lg:items-center lg:justify-between">
            <div className="max-w-2xl">
              <h2 className="text-3xl font-bold tracking-tight">
                Start with the latest release or read the deployment guide.
              </h2>
              <p className="mt-4 text-subtle">
                Download the binary from GitHub Releases, generate a config, and
                run PBLauncher behind your domain. The docs include production
                deployment notes and configuration reference.
              </p>
            </div>
            <div className="mt-8 flex flex-col gap-3 sm:flex-row lg:mt-0">
              <Link
                to="/docs"
                className="inline-flex items-center justify-center gap-2 rounded-xl bg-white px-5 py-3 font-semibold text-background hover:bg-subtle"
              >
                Read docs
                <ArrowRight className="h-4 w-4" />
              </Link>
              <a
                href={RELEASES_URL}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center justify-center gap-2 rounded-xl border border-border px-5 py-3 font-semibold text-foreground hover:border-accent"
              >
                Download
              </a>
            </div>
          </div>
        </section>
      </main>

      <footer className="border-t border-border py-8">
        <div className="mx-auto flex max-w-7xl flex-col gap-4 px-4 text-sm text-muted sm:flex-row sm:items-center sm:justify-between sm:px-6 lg:px-8">
          <p>PBLauncher is an open source project for PocketBase instance management.</p>
          <div className="flex gap-4">
            <a href={GITHUB_URL} target="_blank" rel="noopener noreferrer" className="hover:text-foreground">
              GitHub
            </a>
            <Link to="/docs" className="hover:text-foreground">
              Documentation
            </Link>
          </div>
        </div>
      </footer>
    </div>
  );
};
