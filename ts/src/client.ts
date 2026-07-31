import createFetchClient from "openapi-fetch";
import type { paths } from "./generated/schema.js";

export interface ClientOptions {
  baseUrl: string;
  headers?: HeadersInit;
}

export function createClient(options: ClientOptions) {
  return createFetchClient<paths>({
    baseUrl: options.baseUrl,
    headers: options.headers,
  });
}
