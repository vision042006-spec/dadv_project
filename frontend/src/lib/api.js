// In production, set VITE_API_URL to your Railway backend URL.
// In local dev, requests go through Vite's proxy at /api -> localhost:8080.
const API_BASE = import.meta.env.VITE_API_URL
  ? `${import.meta.env.VITE_API_URL}/api`
  : '/api';

function getAuthHeaders() {
  const token = localStorage.getItem('token');
  return token ? { Authorization: `Bearer ${token}` } : {};
}

async function apiFetch(path, options = {}) {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      ...getAuthHeaders(),
      ...(options.headers || {}),
    },
  });
  return res;
}

export async function uploadFile(file) {
  const formData = new FormData();
  formData.append('file', file);

  const res = await apiFetch('/upload', { method: 'POST', body: formData });
  if (!res.ok) {
    const data = await res.json();
    throw new Error(data.error || 'Upload failed');
  }
  return res.json();
}

export async function login(email, password) {
  const res = await fetch(`${API_BASE}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error);
  return data;
}

export async function signup(email, password, name) {
  const res = await fetch(`${API_BASE}/auth/signup`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password, name }),
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error);
  return data;
}

export async function getMe(token) {
  const res = await fetch(`${API_BASE}/auth/me`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error);
  return data;
}

export async function getJobStatus(jobId) {
  const res = await apiFetch(`/job-status/${jobId}`);
  if (!res.ok) throw new Error('Job not found');
  return res.json();
}

export async function getFileTypeStats(jobId) {
  const res = await apiFetch(`/stats/file-types/${jobId}`);
  if (!res.ok) return { data: [] };
  return res.json();
}

export async function getSizeDistribution(jobId) {
  const res = await apiFetch(`/stats/size-distribution/${jobId}`);
  if (!res.ok) return { data: [] };
  return res.json();
}

export async function getOwnershipStats(jobId) {
  const res = await apiFetch(`/stats/ownership/${jobId}`);
  if (!res.ok) return { data: [] };
  return res.json();
}

export async function getTemporalStats(jobId) {
  const res = await apiFetch(`/stats/temporal/${jobId}`);
  if (!res.ok) return { data: [] };
  return res.json();
}

export async function getAggregateStats(jobId) {
  const res = await apiFetch(`/stats/aggregate/${jobId}`);
  if (!res.ok) return { data: {} };
  return res.json();
}

export async function getAnomalies(jobId) {
  const res = await apiFetch(`/anomalies/${jobId}`);
  if (!res.ok) return { data: [] };
  return res.json();
}