import { useNavigate } from "react-router-dom";
import { authService } from "../services/authService";

function Home() {
  const navigate = useNavigate();

  const handleLogout = () => {
    authService.logout();
    navigate("/login");
  };

  return (
    <div style={{ padding: "40px", maxWidth: "600px", margin: "0 auto" }}>
      <h1>Welcome Home!</h1>
      <p style={{ color: "#666", marginBottom: "20px" }}>
        You are successfully logged in. This page is protected and only
        accessible to authenticated users.
      </p>

      <div
        style={{
          padding: "20px",
          backgroundColor: "#f0f9ff",
          borderRadius: "8px",
          marginBottom: "20px",
        }}
      >
        <h3 style={{ margin: "0 0 10px 0" }}>Route Protection Test</h3>
        <ul style={{ margin: 0, paddingLeft: "20px" }}>
          <li>
            ✅ <strong>ProtectedRoute</strong> is working - you can see this
            page
          </li>
          <li>
            ✅ <strong>PublicRoute</strong> redirected you here from
            login/signup
          </li>
        </ul>
      </div>

      <button
        onClick={handleLogout}
        style={{
          padding: "10px 20px",
          backgroundColor: "#ef4444",
          color: "white",
          border: "none",
          borderRadius: "6px",
          cursor: "pointer",
          fontSize: "16px",
        }}
      >
        Logout
      </button>
    </div>
  );
}

export default Home;
