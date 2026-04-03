// In production we often want same-origin requests (empty base) so `/api/...` is proxied by nginx.
// Using `??` (not `||`) ensures an explicitly empty string stays empty (and does not fall back).
const API_BASE = (import.meta.env.VITE_API_BASE as string | undefined) ?? 'http://localhost:3001';

const TOKEN_KEY = 'gmap_scraper_token';
const TOKEN_EXPIRY_KEY = 'gmap_scraper_token_expiry';

export function getToken(): string | null {
	if (typeof window === 'undefined') return null;

	const token = localStorage.getItem(TOKEN_KEY);
	const expiry = localStorage.getItem(TOKEN_EXPIRY_KEY);

	if (!token || !expiry) return null;

	// Check if token is expired
	if (new Date(expiry) < new Date()) {
		clearToken();
		return null;
	}

	return token;
}

export function setToken(token: string, expiresAt: string): void {
	localStorage.setItem(TOKEN_KEY, token);
	localStorage.setItem(TOKEN_EXPIRY_KEY, expiresAt);
}

export function clearToken(): void {
	localStorage.removeItem(TOKEN_KEY);
	localStorage.removeItem(TOKEN_EXPIRY_KEY);
}

export function isAuthenticated(): boolean {
	return getToken() !== null;
}

async function fetchAPI(path: string, options: RequestInit = {}): Promise<Response> {
	const token = getToken();
	const headers: Record<string, string> = {
		'Content-Type': 'application/json',
		...(options.headers as Record<string, string> || {}),
	};

	if (token) {
		headers['Authorization'] = `Bearer ${token}`;
	}

	const res = await fetch(`${API_BASE}${path}`, {
		...options,
		headers,
	});

	if (res.status === 401) {
		clearToken();
		if (typeof window !== 'undefined') {
			window.location.href = '/login';
		}
	}

	return res;
}

// Generic API Request helper
export async function apiRequest(path: string, method: string = 'GET', body?: any): Promise<any> {
	const options: RequestInit = { method };
	if (body) {
		options.body = JSON.stringify(body);
	}

	const res = await fetchAPI(path, options);
	if (!res.ok) {
		const data = await res.json().catch(() => ({}));
		throw new Error(data.error || `Request failed with status ${res.status}`);
	}

	if (res.status === 204) return null;
	return res.json();
}

// Auth
export async function login(password: string): Promise<{ token: string; expires_at: string }> {
	const res = await fetch(`${API_BASE}/api/auth/login`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ password }),
	});

	if (!res.ok) {
		const data = await res.json();
		throw new Error(data.error || 'Login failed');
	}

	const data = await res.json();
	setToken(data.token, data.expires_at);
	return data;
}

// Jobs
export async function createScrapeJob(req: any): Promise<any> {
	const res = await fetchAPI('/api/scrape', {
		method: 'POST',
		body: JSON.stringify(req),
	});

	if (!res.ok) {
		const data = await res.json();
		throw new Error(data.error || 'Failed to create job');
	}

	return res.json();
}

export async function listJobs(): Promise<any[]> {
	const res = await fetchAPI('/api/jobs');
	if (!res.ok) throw new Error('Failed to fetch jobs');
	return res.json();
}

export async function getJob(id: string): Promise<any> {
	const res = await fetchAPI(`/api/jobs/${id}`);
	if (!res.ok) throw new Error('Job not found');
	return res.json();
}

export async function deleteJob(id: string): Promise<void> {
	const res = await fetchAPI(`/api/jobs/${id}`, { method: 'DELETE' });
	if (!res.ok) throw new Error('Failed to delete job');
}

export async function getPlaces(jobId: string): Promise<any[]> {
	const res = await fetchAPI(`/api/jobs/${jobId}/places`);
	if (!res.ok) throw new Error('Failed to fetch places');
	return res.json();
}

// SSE Stream for real-time updates
export function streamJob(jobId: string, onEvent: (event: any) => void): () => void {
	const token = getToken();
	const url = `${API_BASE}/api/jobs/${jobId}/stream?token=${encodeURIComponent(token || '')}`;

	const eventSource = new EventSource(url);

	eventSource.onmessage = (event) => {
		try {
			const data = JSON.parse(event.data);
			onEvent(data);

			if (data.type === 'done' || data.type === 'error') {
				eventSource.close();
			}
		} catch {
			// ignore parse errors
		}
	};

	eventSource.onerror = () => {
		eventSource.close();
	};

	// Return cleanup function
	return () => eventSource.close();
}
