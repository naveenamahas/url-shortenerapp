import { useEffect, useState, useCallback } from "react";
import { api } from "../api";
import LinkRow from "./LinkRow.jsx";

export default function Dashboard({ onLogout }) {
  const [links, setLinks] = useState([]);
  const [loading, setLoading] = useState(true);
  const [url, setUrl] = useState("");
  const [alias, setAlias] = useState("");
  const [expires, setExpires] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      const data = await api("/api/links");
      setLinks(data.links || []);
    } catch (err) {
      if (err.status === 401) onLogout(); // token expired
    } finally {
      setLoading(false);
    }
  }, [onLogout]);

  useEffect(() => {
    load();
    const id = setInterval(load, 5000); // refresh click counts every 5s
    return () => clearInterval(id);
  }, [load]);

  async function create(e) {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      const body = { url };
      if (alias.trim()) body.alias = alias.trim();
      if (expires) body.expiresAt = new Date(expires).toISOString();
      await api("/api/links", { method: "POST", body });
      setUrl(""); setAlias(""); setExpires("");
      await load();
    } catch (err) {
      setError(err.message); // shows 400 / 409 / 429 messages from the server
    } finally {
      setBusy(false);
    }
  }

  async function remove(code) {
    if (!confirm("Delete this link? The short URL will stop working.")) return;
    await api(`/api/links/${code}`, { method: "DELETE" });
    load();
  }

  const totalClicks = links.reduce((sum, l) => sum + l.clicks, 0);

  return (
    <main>
      <form className="card" onSubmit={create}>
        <h2>Shorten a link</h2>
        <input
          className="big"
          placeholder="Paste a long URL, e.g. https://example.com/very/long/path"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          required
        />
        <div className="row">
          <div>
            <label>Custom alias (optional)</label>
            <input placeholder="my-link" value={alias} onChange={(e) => setAlias(e.target.value)} />
          </div>
          <div>
            <label>Expires (optional)</label>
            <input type="datetime-local" value={expires} onChange={(e) => setExpires(e.target.value)} />
          </div>
        </div>
        {error && <div className="error">{error}</div>}
        <button className="btn" disabled={busy}>{busy ? "Shortening..." : "Shorten"}</button>
      </form>

      <div className="stats">
        <div className="card stat"><b>{links.length}</b><span>Links</span></div>
        <div className="card stat"><b>{totalClicks}</b><span>Total clicks</span></div>
      </div>

      <section className="card">
        <h2>Your links</h2>
        {loading && <p className="muted">Loading...</p>}
        {!loading && links.length === 0 && <p className="muted">No links yet. Create your first one above.</p>}
        {links.map((l) => (
          <LinkRow key={l.code} link={l} onDelete={() => remove(l.code)} />
        ))}
      </section>
    </main>
  );
}
