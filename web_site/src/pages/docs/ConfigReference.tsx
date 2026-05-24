export const ConfigReference = () => (
  <section className="overflow-hidden rounded-2xl bg-white p-4 shadow-xl sm:p-6">
    <h2 className="text-2xl font-bold mb-5">Config Reference</h2>

    <h3 className="text-lg font-semibold mb-3">Network settings</h3>
    <ul className="list-disc ml-6 space-y-2 text-sm">
      <li>
        <span className="font-mono">domain</span>: Public domain to be used for
        the service.
      </li>
      <li>
        <span className="font-mono">bind_address</span>:{" "}
        <b>IPv4 loopback address</b> (e.g.{" "}
        <span className="font-mono">127.0.0.1</span>), used only for internal
        PocketBase instances. <br />
        <span className="text-yellow-700 font-medium">
          Warning: Never set to your public IP or{" "}
          <span className="font-mono">0.0.0.0</span>.
        </span>
      </li>
      <li>
        <span className="font-mono">listen_address</span>: Address where
        pb_launcher listens for external connections (usually your public IP or{" "}
        <span className="font-mono">0.0.0.0</span>).
      </li>
      <li>
        <span className="font-mono">http_port</span>: Port for HTTP traffic
        (e.g. <span className="font-mono">7080</span> or{" "}
        <span className="font-mono">80</span>).
      </li>
      <li>
        <span className="font-mono">https</span>: Enable (
        <span className="font-mono">true</span>) or disable (
        <span className="font-mono">false</span>) HTTPS.
      </li>
      <li>
        <span className="font-mono">https_port</span>: Port for HTTPS traffic
        (e.g. <span className="font-mono">8443</span> or{" "}
        <span className="font-mono">443</span>).
      </li>
      <li>
        <span className="font-mono">disable_https_redirect</span>: If{" "}
        <span className="font-mono">false</span>, HTTP will redirect to HTTPS
        automatically.
      </li>
    </ul>

    <h3 className="text-lg font-semibold mt-6 mb-3">Paths</h3>
    <ul className="list-disc ml-6 space-y-2 text-sm">
      <li>
        <span className="font-mono">download_dir</span>: Directory to download
        PocketBase binaries.
      </li>
      <li>
        <span className="font-mono">certificates_dir</span>: Directory to store
        SSL certificates.
      </li>
      <li>
        <span className="font-mono">accounts_dir</span>: Let's Encrypt accounts
        directory (for ACME).
      </li>
      <li>
        <span className="font-mono">data_dir</span>: Directory to store
        PocketBase instance data.
      </li>
    </ul>

    <h3 className="text-lg font-semibold mt-6 mb-3">Certificate management</h3>
    <ul className="list-disc ml-6 space-y-2 text-sm">
      <li>
        <span className="font-mono">acme_email</span>: Email address required
        for Let's Encrypt/ACME certificates (required if HTTPS is enabled).
      </li>
      <li>
        <span className="font-mono">min_certificate_ttl</span>: Minimum
        certificate validity before attempting renewal (e.g.{" "}
        <span className="font-mono">720h</span>).
      </li>
      <li>
        <span className="font-mono">max_domain_cert_attempts</span>: Maximum
        number of attempts to obtain a certificate before marking as failed.
      </li>
      <li>
        <span className="font-mono">cert_request_planner_interval</span>:
        Interval for checking and scheduling certificate renewal tasks (for
        custom HTTP-01 challenges).
      </li>
      <li>
        <span className="font-mono">cert_request_executor_interval</span>:
        Interval for executing scheduled certificate renewal tasks.
      </li>
      <li>
        <span className="font-mono">certificate_check_interval</span>: Interval
        to check the status of the main (wildcard) certificate.
      </li>
      <li>
        <span className="font-mono">cert.provider</span>: DNS provider for
        certificate issuance. Supported:{" "}
        <span className="font-mono">selfsigned</span>,{" "}
        <span className="font-mono">mkcert</span>,{" "}
        <span className="font-mono">cloudflare</span>.
      </li>
      <li>
        <span className="font-mono">cert.props.auth_token</span>: (Cloudflare
        only) API token with DNS edit permissions.
      </li>
    </ul>

    <h3 className="text-lg font-semibold mt-6 mb-3">Sync & command checks</h3>
    <ul className="list-disc ml-6 space-y-2 text-sm">
      <li>
        <span className="font-mono">release_sync_interval</span>: Interval for
        checking new PocketBase releases on GitHub (e.g.{" "}
        <span className="font-mono">5m</span>).
      </li>
      <li>
        <span className="font-mono">command_check_interval</span>: Interval for
        processing commands such as start, stop, or restart instances (e.g.{" "}
        <span className="font-mono">10s</span>).
      </li>
    </ul>

    <div className="mt-8 text-xs text-gray-600">
      For details and updates, see the{" "}
      <a
        href="https://github.com/user0608/pb_launcher"
        className="underline text-blue-700"
        target="_blank"
        rel="noopener noreferrer"
      >
        official pb_launcher repository
      </a>
      .
    </div>
    <div className="mt-10">
      <h3 className="text-lg font-semibold mb-2">
        Example <span className="font-mono">config.yml</span>
      </h3>
      <pre className="bg-gray-800 text-green-200 rounded px-4 py-3 text-xs font-mono overflow-x-auto leading-relaxed">
        {`domain: pb.labenv.test

bind_address: 127.0.0.1

listen_address: 0.0.0.0
http_port: "7080"

https: false
https_port: "8443"

disable_https_redirect: false

download_dir: ./downloads
certificates_dir: ./.certificates
accounts_dir: ./.accounts
data_dir: ./data

acme_email: ""
min_certificate_ttl: 720h
max_domain_cert_attempts: 1
cert_request_planner_interval: 5m
cert_request_executor_interval: 1m
certificate_check_interval: 1m

# cert:
#   provider: "cloudflare"
#   props:
#     auth_token: ""

release_sync_interval: 5m
command_check_interval: 10s`}
      </pre>
    </div>
  </section>
);
