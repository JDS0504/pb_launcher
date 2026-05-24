import { Link } from "react-router";

export const Instructions = () => (
  <section className="overflow-hidden rounded-2xl bg-white p-4 shadow-xl sm:p-6">
    <h2 className="mb-5 text-2xl font-bold">Production Deployment Guide</h2>
    <ol className="list-decimal list-inside space-y-4 text-base">
      <DownloadRelease />
      <ExtractAndMoveFiles />
      <DNSSetup />
      <GenerateAndEditConfig />
      <SystemdService />
      <EnableAndStartService />
      <DNSProviderSetup />
      <Notes />
    </ol>
  </section>
);

// region: DownloadRelease
const DownloadRelease = () => (
  <li>
    <span className="font-semibold">Download the latest </span>
    <span className="font-mono px-1">pb_launcher</span>
    <span> release for your architecture from the </span>
    <a
      href="https://github.com/user0608/pb_launcher/releases"
      target="_blank"
      rel="noopener noreferrer"
      className="text-blue-600 underline font-medium"
    >
      releases page
    </a>
    <span>. For example:</span>
    <pre className="bg-gray-100 rounded px-3 py-2 mt-2 mb-0 text-sm text-gray-800 font-mono overflow-x-auto">
      wget
      https://github.com/user0608/pb_launcher/releases/download/&lt;version&gt;/pb_launcher_&lt;version&gt;_linux_amd64.zip
    </pre>
  </li>
);

// region: ExtractAndMoveFiles
const ExtractAndMoveFiles = () => (
  <li>
    <span className="font-semibold">Extract and move files:</span>
    <pre className="bg-gray-100 rounded px-3 py-2 mt-2 mb-0 text-sm text-gray-800 font-mono overflow-x-auto">
      {`sudo mkdir -p /opt/pb_launcher
sudo unzip pb_launcher_<version>_linux_amd64.zip -d /opt/pb_launcher
cd /opt/pb_launcher`}
    </pre>
  </li>
);

// region: DNSSetup
const DNSSetup = () => (
  <li>
    <span className="font-semibold">DNS setup:</span>
    <div className="mt-2 ml-1 space-y-3">
      <p>
        You must configure both your base domain and a wildcard subdomain to
        point to your server's public IP address.
      </p>
      <ul className="list-disc ml-7 text-sm space-y-1">
        <li>
          <span className="font-mono">A</span> record for your main domain
          (e.g., <span className="font-mono">yourdomain.com</span>).
        </li>
        <li>
          <span className="font-mono">A</span> record for the wildcard domain
          (e.g., <span className="font-mono">*.yourdomain.com</span>).
        </li>
      </ul>
      <div className="bg-blue-50 border-l-4 border-blue-400 p-3 rounded text-sm text-blue-800">
        <strong>What is a wildcard domain?</strong>
        <br />A wildcard DNS record allows you to point all subdomains (
        <span className="font-mono">anything.yourdomain.com</span>) to a single
        IP address, without creating each record manually. This is required for
        pb_launcher to dynamically assign subdomains.
      </div>
      <div className="bg-yellow-50 border-l-4 border-yellow-400 p-3 rounded text-sm text-yellow-800">
        <strong>Note:</strong>
        <br />
        The DNS configuration and requirements may vary depending on the
        selected DNS provider. Please review the{" "}
        <span className="font-mono">cert.provider</span> options available in
        the documentation and ensure you follow the setup steps for your chosen
        provider.
      </div>
      <div className="mt-3">
        <span>
          For more on wildcard DNS, see{" "}
          <a
            href="https://developers.cloudflare.com/dns/manage-dns-records/reference/wildcard-dns-records/"
            className="underline text-blue-700"
            target="_blank"
            rel="noopener noreferrer"
          >
            Cloudflare: Wildcard DNS Records
          </a>
          .
        </span>
      </div>
    </div>
  </li>
);

// region: GenerateAndEditConfig
const GenerateAndEditConfig = () => (
  <li>
    <span className="font-semibold">Generate and edit the </span>
    <span className="font-mono px-1">config.yml</span>
    <span> file:</span>
    <ol className="list-decimal ml-7 mt-3 space-y-2 text-sm">
      <li>
        Run the following command to generate the configuration file:
        <pre className="bg-gray-800 text-green-200 rounded px-3 py-2 mt-2 text-sm font-mono overflow-x-auto">
          {`./pblauncher gen-config > config.yml`}
        </pre>
      </li>
      <li>
        Open <span className="font-mono">config.yml</span> with your preferred
        editor:
        <pre className="bg-gray-800 text-green-200 rounded px-3 py-2 mt-2 text-sm font-mono overflow-x-auto">
          vim config.yml
        </pre>
        Modify <span className="font-semibold">at least</span> the following
        fields:
        <ul className="list-disc ml-7 mt-3 space-y-2 text-sm">
          <li>
            <span className="font-mono">domain</span>: your domain, e.g.{" "}
            <span className="font-mono">yourdomain.com</span>
            <span className="text-gray-600"> (required)</span>
          </li>
          <li>
            <span className="font-mono">http_port</span>:{" "}
            <span className="font-mono">80</span>
          </li>
          <li>
            <span className="font-mono">https</span>:{" "}
            <span className="font-mono">true</span>
          </li>
          <li>
            <span className="font-mono">https_port</span>:{" "}
            <span className="font-mono">443</span>
          </li>
          <li>
            <span className="font-mono">acme_email</span>: a valid email address
            for Let's Encrypt
            <span className="text-gray-600"> (required)</span>
          </li>
        </ul>
      </li>
    </ol>
    <div className="mt-4">
      For details, see the{" "}
      <Link to="/docs/config" className="text-blue-600 underline font-medium">
        Config Reference
      </Link>
      .
    </div>
  </li>
);

// region: SystemdService
const SystemdService = () => (
  <li>
    <span className="font-semibold">Create a systemd service:</span>
    <pre className="bg-gray-100 rounded px-3 py-2 mt-2 mb-0 text-sm text-gray-800 font-mono overflow-x-auto">
      sudo vim /etc/systemd/system/pblauncher.service
    </pre>
    <div className="mt-1 text-xs text-gray-700">
      <span>
        Use the following template (adjust if your path is different):
      </span>
    </div>
    <pre className="bg-gray-900 text-blue-100 rounded px-3 py-2 mt-2 text-xs font-mono overflow-x-auto">
      {`[Unit]
Description=PB Launcher Service
After=network.target

[Service]
WorkingDirectory=/opt/pb_launcher
ExecStart=/opt/pb_launcher/pblauncher -c config.yml
Restart=on-failure
User=root
Group=root

[Install]
WantedBy=multi-user.target`}
    </pre>
  </li>
);

// region: EnableAndStartService
const EnableAndStartService = () => (
  <li>
    <span className="font-semibold">Enable and start the service:</span>
    <pre className="bg-gray-100 rounded px-3 py-2 mt-2 mb-0 text-sm text-gray-800 font-mono overflow-x-auto">
      {`sudo systemctl daemon-reload
sudo systemctl enable pblauncher.service
sudo systemctl start pblauncher.service
sudo systemctl status pblauncher.service`}
    </pre>
  </li>
);

// region: DNSProviderSetup
const DNSProviderSetup = () => (
  <li>
    <span className="font-semibold">DNS Provider setup:</span>
    <div className="mt-2 text-sm">
      <p>
        Set the <span className="font-mono">cert.provider</span> field in your{" "}
        <span className="font-mono">config.yml</span> to choose the DNS provider
        for certificate management. Each provider may require different options
        in <span className="font-mono">cert.props</span>:
      </p>
      <div className="overflow-x-auto mt-4">
        <table className="min-w-full text-left border border-gray-200 rounded-lg bg-white">
          <thead>
            <tr className="bg-gray-100">
              <th className="px-4 py-2 border-b font-semibold">Provider</th>
              <th className="px-4 py-2 border-b font-semibold">Description</th>
              <th className="px-4 py-2 border-b font-semibold">
                Required <span className="font-mono">props</span>
              </th>
              <th className="px-4 py-2 border-b font-semibold">Notes</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td className="px-4 py-2 border-b align-top">
                <span className="font-mono">selfsigned</span>
              </td>
              <td className="px-4 py-2 border-b align-top">
                Generates a self-signed certificate for local
                development/testing.
              </td>
              <td className="px-4 py-2 border-b align-top">None</td>
              <td className="px-4 py-2 border-b align-top text-gray-500">
                Not recommended for production.
              </td>
            </tr>
            <tr>
              <td className="px-4 py-2 border-b align-top">
                <span className="font-mono">mkcert</span>
              </td>
              <td className="px-4 py-2 border-b align-top">
                Uses{" "}
                <a
                  href="https://github.com/FiloSottile/mkcert"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="underline text-blue-700"
                >
                  mkcert
                </a>{" "}
                to generate trusted certificates locally.
              </td>
              <td className="px-4 py-2 border-b align-top">None</td>
              <td className="px-4 py-2 border-b align-top text-gray-500">
                You must have <span className="font-mono">mkcert</span>{" "}
                installed on your server.
              </td>
            </tr>
            <tr>
              <td className="px-4 py-2 border-b align-top">
                <span className="font-mono">cloudflare</span>
              </td>
              <td className="px-4 py-2 border-b align-top">
                Uses Cloudflare DNS-01 challenge for Let's Encrypt wildcard
                certificates.
              </td>
              <td className="px-4 py-2 border-b align-top">
                <div>
                  <span className="font-mono">auth_token</span>{" "}
                  <span className="text-gray-500">(string)</span>
                </div>
              </td>
              <td className="px-4 py-2 border-b align-top">
                API token must have <span className="font-mono">Zone.DNS</span>{" "}
                edit permission.
                <br />
                Recommended for production.
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div className="mt-6">
        <div className="mb-2 text-gray-800 font-semibold">
          Example <span className="font-mono">cert</span> section for
          Cloudflare:
        </div>
        <pre className="bg-gray-800 text-green-200 rounded px-4 py-3 font-mono text-sm overflow-x-auto">
          <code>
            {`cert:
  provider: "cloudflare"
  props:
    auth_token: "your_cloudflare_api_token"`}
          </code>
        </pre>
        <div className="mt-2 text-xs text-gray-600">
          Replace <span className="font-mono">your_cloudflare_api_token</span>{" "}
          with your actual Cloudflare API token. The token must have permissions
          to manage DNS for your zone.
        </div>
      </div>
      <div className="mt-4 bg-yellow-50 border-l-4 border-yellow-400 p-3 rounded text-yellow-800">
        <strong>Note:</strong> More providers may be added in the future. Each
        provider may require specific props—always refer to the documentation
        for details.
      </div>
    </div>
  </li>
);

// region: Notes
const Notes = () => (
  <li>
    <span className="font-semibold">Notes:</span>
    <ul className="list-disc list-inside ml-6 mt-2 text-sm">
      <li>
        <span className="font-mono">bind_address</span> should always be set to
        <span className="font-mono px-1">127.0.0.1</span> or any IPv4 loopback
        address (e.g. <span className="font-mono">127.x.x.x</span>) for internal
        PocketBase instances.
        <div className="mt-2 bg-yellow-50 border-l-4 border-yellow-400 px-4 py-2 rounded text-yellow-800 font-medium">
          <strong>Warning:</strong> <br />
          Never set <span className="font-mono">bind_address</span> to your
          public IP or <span className="font-mono">0.0.0.0</span>. This would
          expose PocketBase instances to the network and create a serious
          security risk. Always use a loopback IPv4 address such as{" "}
          <span className="font-mono">127.0.0.1</span>.
        </div>
      </li>
      <li>
        <span className="font-mono">listen_address</span> should be your public
        IP (or <span className="font-mono">0.0.0.0</span>).
      </li>
      <li>Never expose PocketBase instances directly to the public network.</li>
    </ul>
  </li>
);
