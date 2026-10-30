import { API_BASE_URL } from '$app/env/private';
import { proxyApi } from '#lib/server/api-proxy.js';
import type { RequestHandler } from './$types';

export const config = { runtime: 'nodejs22.x', maxDuration: 60, split: true };

const proxy: RequestHandler = ({ request, url }) => proxyApi(request, url, API_BASE_URL);
export const GET = proxy;
export const HEAD = proxy;
export const POST = proxy;
export const PATCH = proxy;
export const PUT = proxy;
export const DELETE = proxy;
