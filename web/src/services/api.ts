// Base API configuration and client

const API_BASE_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";

interface ApiResponse<T> {
  data?: T;
  error?: string;
}

class ApiClient {
  private baseUrl: string;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  private getHeaders(): HeadersInit {
    const headers: HeadersInit = {
      "Content-Type": "application/json",
    };

    // Add auth token if available
    const token = localStorage.getItem("token");
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }

    return headers;
  }

  // Handle 401 Unauthorized (token expired or invalid)
  private handleUnauthorized() {
    localStorage.removeItem("token");
    // Redirect to login with expired flag if not already there
    if (window.location.pathname !== "/login") {
      window.location.href = "/login?expired=true";
    }
  }

  async get<T>(endpoint: string): Promise<ApiResponse<T>> {
    try {
      const response = await fetch(`${this.baseUrl}${endpoint}`, {
        method: "GET",
        headers: this.getHeaders(),
      });

      if (response.status === 401) {
        this.handleUnauthorized();
        return { error: "Session expired. Please login again." };
      }

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        return { error: errorData.message || `Error: ${response.status}` };
      }

      const data = await response.json();
      return { data };
    } catch (error) {
      return {
        error: error instanceof Error ? error.message : "Network error",
      };
    }
  }

  async post<T>(endpoint: string, body: unknown): Promise<ApiResponse<T>> {
    try {
      const response = await fetch(`${this.baseUrl}${endpoint}`, {
        method: "POST",
        headers: this.getHeaders(),
        body: JSON.stringify(body),
      });

      if (response.status === 401) {
        this.handleUnauthorized();
        return { error: "Session expired. Please login again." };
      }

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        return { error: errorData.message || `Error: ${response.status}` };
      }

      const data = await response.json();
      return { data };
    } catch (error) {
      return {
        error: error instanceof Error ? error.message : "Network error",
      };
    }
  }

  async put<T>(endpoint: string, body: unknown): Promise<ApiResponse<T>> {
    try {
      const response = await fetch(`${this.baseUrl}${endpoint}`, {
        method: "PUT",
        headers: this.getHeaders(),
        body: JSON.stringify(body),
      });

      if (response.status === 401) {
        this.handleUnauthorized();
        return { error: "Session expired. Please login again." };
      }

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        return { error: errorData.message || `Error: ${response.status}` };
      }

      const data = await response.json();
      return { data };
    } catch (error) {
      return {
        error: error instanceof Error ? error.message : "Network error",
      };
    }
  }

  async delete<T>(endpoint: string): Promise<ApiResponse<T>> {
    try {
      const response = await fetch(`${this.baseUrl}${endpoint}`, {
        method: "DELETE",
        headers: this.getHeaders(),
      });

      if (response.status === 401) {
        this.handleUnauthorized();
        return { error: "Session expired. Please login again." };
      }

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        return { error: errorData.message || `Error: ${response.status}` };
      }

      const data = await response.json();
      return { data };
    } catch (error) {
      return {
        error: error instanceof Error ? error.message : "Network error",
      };
    }
  }
}

export const apiClient = new ApiClient(API_BASE_URL);
export type { ApiResponse };
