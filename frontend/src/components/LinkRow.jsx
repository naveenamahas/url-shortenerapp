import { useState } from "react";
import { api } from "../api";

export default function LinkRow({ link, onDelete }) {
  const [copied, setCopied] = useState(false);
  const [days, setDays] = useState(null); // null = chart closed

  async function copy() {
    await navigator.clipboard.writeText(link.shortUrl);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }

  async function toggleChart() {
    if (days) return setDays(null);
    const data = await api(`/api/links/${link.code}/stats`);
    setDays(data.days);
  }

  const expired = link.expiresAt && new Date(link.expiresAt) < new Date();
  const max = days ? Math.max(1, ...days.map((d) => d.clicks)) : 1;

  return (
    <div className="link">
      <div className="link-main">
        <a href={link.shortUrl} target="_blank" rel="noreferrer" className="short">{link.shortUrl}</a>
        <div className="muted ellipsis" title={link.url}>{link.url}</div>
        {link.expiresAt && (
          <div className={expired ? "tag danger" : "tag"}>
            {expired ? "Expired" : "Expires"} {new Date(link.expiresAt).toLocaleString()}
          </div>
        )}
      </div>
      <div className="clicks"><b>{link.clicks}</b><span>clicks</span></div>
      <div className="actions">
        <button className="btn small" onClick={copy}>{copied ? "Copied!" : "Copy"}</button>
        <button className="btn small ghost" onClick={toggleChart}>{days ? "Hide" : "Chart"}</button>
        <button className="btn small danger" onClick={onDelete}>Delete</button>
      </div>
      {days && (
        <div className="chart">
          {days.map((d) => (
            <div key={d.date} className="bar-col" title={`${d.date}: ${d.clicks} clicks`}>
              <span className="bar-num">{d.clicks}</span>
              <div className="bar" style={{ height: `${(d.clicks / max) * 80 + 2}px` }} />
              <span className="bar-day">{d.date.slice(5)}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
