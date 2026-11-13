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

  return (
    <form className="card auth" onSubmit={submit}>
      <h2>{mode === "login" ? "Welcome back" : "Create your account"}</h2>
      <p className="muted">Shorten links and see who clicks them.</p>
      <label>Email</label>
      <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} required autoFocus />
      <label>Password</label>
      <input
        type="password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        minLength={8}
        required
        placeholder="At least 8 characters"
      />
      {error && <div className="error">{error}</div>}
      <button className="btn" disabled={busy}>
        {busy ? "Please wait..." : mode === "login" ? "Log in" : "Sign up"}
      </button>
      <p className="muted center">
        {mode === "login" ? "New here?" : "Already have an account?"}{" "}
        <a href="#" onClick={(e) => { e.preventDefault(); setError(""); setMode(mode === "login" ? "signup" : "login"); }}>
          {mode === "login" ? "Sign up" : "Log in"}
        </a>
      </p>
    </form>
  );
}
