import { useState } from "react";
import { getToken, clearToken } from "./api";
import Auth from "./components/Auth.jsx";
import Dashboard from "./components/Dashboard.jsx";

export default function App() {
  const [loggedIn, setLoggedIn] = useState(!!getToken());

  const logout = () => {
    clearToken();
    setLoggedIn(false);
  };

  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="brand-wrap">
          <div className="brand-mark">S</div>
          <div className="brand-copy">
            <span className="eyebrow">URL shortener</span>
            <h1>Snip</h1>
          </div>
        </div>
        {loggedIn && (
          <button className="btn ghost" onClick={logout}>
            Log out
          </button>
        )}
      </header>

      {!loggedIn && (
        <div className="hero-panel">
          <div>
            <span className="badge">Smarter links</span>
            <h2>Shorten, track, and share with confidence.</h2>
          </div>
          <p>Turn long URLs into polished links your audience will actually click.</p>
        </div>
      )}

      {loggedIn ? <Dashboard onLogout={logout} /> : <Auth onDone={() => setLoggedIn(true)} />}
    </div>
  );
}
