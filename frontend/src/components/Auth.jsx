import { useState } from "react";
import { api, setToken } from "../api";

export default function Auth({ onDone }) {
  const [mode, setMode] = useState("login"); // "login" | "signup"
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(e) {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      const data = await api(`/api/auth/${mode}`, { method: "POST", body: { email, password } });
      setToken(data.token);
      onDone();
    } catch (err) {
      setError(err.message);
    } finally {
      setBusy(false);
    }
  }

  const isLogin = mode === "login";

  return (
    <form className="card auth" onSubmit={submit}>
      <div className="auth-mark" aria-hidden="true">S</div>
      <div className="auth-heading">
        <span className="eyebrow">{isLogin ? "Welcome back" : "Get started"}</span>
        <h2>{isLogin ? "Sign in to Snip" : "Create your account"}</h2>
        <p className="muted">
          {isLogin
            ? "Pick up where you left off and keep your links moving."
            : "Create an account to shorten links and follow every click."}
        </p>
      </div>

      <div className="auth-field">
        <label htmlFor="auth-email">Email address</label>
        <input
          id="auth-email"
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          autoComplete="email"
          placeholder="you@example.com"
          required
          autoFocus
        />
      </div>
      <div className="auth-field">
        <label htmlFor="auth-password">Password</label>
        <input
          id="auth-password"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          autoComplete={isLogin ? "current-password" : "new-password"}
          minLength={8}
          required
          placeholder="At least 8 characters"
        />
      </div>
      {error && <div className="error" role="alert">{error}</div>}
      <button className="btn auth-submit" disabled={busy}>
        {busy ? "Please wait..." : isLogin ? "Sign in" : "Create account"}
        {!busy && <span aria-hidden="true">→</span>}
      </button>
      <div className="auth-switch">
        <span className="muted">{isLogin ? "New to Snip?" : "Already have an account?"}</span>
        <button
          className="auth-switch-button"
          type="button"
          onClick={() => {
            setError("");
            setMode(isLogin ? "signup" : "login");
          }}
        >
          {isLogin ? "Create an account" : "Sign in"}
        </button>
      </div>
    </form>
  );
}
