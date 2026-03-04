import { apiClient } from "./api";

// Request/Response types
export interface LoginRequest {
  username: string;
  password: string;
}

export interface SignupRequest {
  type: "email" | "google";
  username: string;
  passwd?: string; // required for email signup
  code?: string; // required for google signup
}

// Backend response format
export interface BackendResponse<T = unknown> {
  success: boolean;
  message?: string;
  data?: T;
}

export interface TokenData {
  token: string;
}

// Auth service
export const authService = {
  async login(credentials: LoginRequest) {
    const response = await apiClient.post<BackendResponse<TokenData>>(
      "/login",
      credentials,
    );

    if (response.data?.success && response.data?.data?.token) {
      localStorage.setItem("token", response.data.data.token);
    }

    // Return error message from backend if not successful
    if (response.data && !response.data.success) {
      return { error: response.data.message || "Login failed" };
    }

    return response;
  },

  async signup(data: SignupRequest) {
    const response = await apiClient.post<BackendResponse<TokenData>>(
      "/signup",
      data,
    );

    if (response.data?.success && response.data?.data?.token) {
      localStorage.setItem("token", response.data.data.token);
    }

    // Return error message from backend if not successful
    if (response.data && !response.data.success) {
      return { error: response.data.message || "Signup failed" };
    }

    return response;
  },

  logout() {
    localStorage.removeItem("token");
  },

  getToken() {
    return localStorage.getItem("token");
  },

  isAuthenticated() {
    return !!this.getToken();
  },
};
