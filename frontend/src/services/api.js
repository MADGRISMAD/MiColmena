// Cliente de la API de MiColmena.
// En desarrollo, Vite redirige /api al backend (ver vite.config.js).
// En producción, define VITE_API_URL con la URL del backend (por ejemplo https://api.micolmena.com).
const BASE_URL = (import.meta.env.VITE_API_URL || '') + '/api';
const TOKEN_KEY = 'micolmena_token';

export function getToken() {
  return localStorage.getItem(TOKEN_KEY);
}

export function setToken(token) {
  if (token) localStorage.setItem(TOKEN_KEY, token);
  else localStorage.removeItem(TOKEN_KEY);
}

async function request(method, path, body) {
  const headers = { 'Content-Type': 'application/json' };
  const token = getToken();
  if (token) headers.Authorization = `Bearer ${token}`;

  const res = await fetch(BASE_URL + path, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body)
  });

  const data = res.status === 204 ? null : await res.json().catch(() => null);
  if (!res.ok) {
    if (res.status === 401) setToken(null);
    const error = new Error(data?.error || `Error ${res.status}`);
    error.status = res.status;
    error.fields = data?.fields;
    throw error;
  }
  return data;
}

function query(params = {}) {
  const entries = Object.entries(params).filter(([, v]) => v !== undefined && v !== null && v !== '');
  return entries.length ? '?' + new URLSearchParams(entries) : '';
}

export const api = {
  async login(email, password) {
    const data = await request('POST', '/auth/login', { email, password });
    setToken(data.token);
    return data.user;
  },
  async register(name, email, password) {
    const data = await request('POST', '/auth/register', { name, email, password });
    setToken(data.token);
    return data.user;
  },
  logout() {
    setToken(null);
  },
  me: () => request('GET', '/me'),

  // filtros: { status, priority, assignee, q, before, limit }
  listTickets: (filters) => request('GET', '/tickets' + query(filters)),
  getTicket: (id) => request('GET', `/tickets/${id}`),
  createTicket: (ticket) => request('POST', '/tickets', ticket),
  updateTicket: (id, changes) => request('PATCH', `/tickets/${id}`, changes),

  listComments: (ticketId) => request('GET', `/tickets/${ticketId}/comments`),
  addComment: (ticketId, body, internal = false) =>
    request('POST', `/tickets/${ticketId}/comments`, { body, internal }),

  listUsers: (role) => request('GET', '/users' + query({ role })),
  setUserRole: (id, role) => request('PATCH', `/users/${id}/role`, { role }),
  stats: () => request('GET', '/stats')
};
