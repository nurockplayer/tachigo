/* eslint-disable */
// Generated from services/api/docs/swagger.json. Do not edit by hand.

export interface HandlersAddressResponse {
  address?: ModelsShippingAddress;
}

export interface HandlersAddressesResponse {
  addresses?: Array<ModelsShippingAddress>;
}

export interface HandlersAuthResponse {
  tokens?: HandlersBrowserTokenPair;
  user?: ModelsUser;
}

export interface HandlersBrowserTokenPair {
  access_token: string;
  expires_in: number;
}

export interface HandlersMessageResponse {
  message?: string;
}

export interface HandlersNonceResponse {
  issued_at?: string;
  nonce?: string;
}

export interface HandlersPointsBalanceResponse {
  cumulative_total?: number;
  spendable_balance?: number;
}

export interface HandlersPointsHistoryItem {
  amount?: number;
  created_at?: string;
  note?: string;
  sku?: string;
  type?: "earn" | "spend";
}

export interface HandlersPointsHistoryResponse {
  transactions?: Array<HandlersPointsHistoryItem>;
}

export interface HandlersProvidersResponse {
  providers?: Array<ModelsAuthProvider>;
}

export interface HandlersResponse {
  data?: unknown;
  error?: string;
  success?: boolean;
}

export interface HandlersTokensResponse {
  tokens?: HandlersBrowserTokenPair;
}

export interface HandlersUserResponse {
  user?: ModelsUser;
}

export interface HandlersWalletResponse {
  address?: string;
}

export interface HandlersClaimRequest {
  amount?: number;
}

export interface HandlersRedeemRequest {
  amount: number;
  coupon_id: string;
}

export interface HandlersRedeemResponse {
  balance?: number;
  voucher_code?: string;
}

export interface HandlersTachiBalanceResponse {
  tachi_balance?: number;
}

export interface ModelsAuthProvider {
  created_at?: string;
  id?: string;
  metadata?: Record<string, unknown>;
  provider?: ModelsProviderType;
  provider_id?: string;
  updated_at?: string;
  user_id?: string;
}

export type ModelsProviderType = "twitch" | "google" | "web3" | "email";

export interface ModelsShippingAddress {
  address_line1?: string;
  address_line2?: string;
  city?: string;
  country?: string;
  created_at?: string;
  district?: string;
  id?: string;
  is_default?: boolean;
  phone?: string;
  postal_code?: string;
  recipient_name?: string;
  updated_at?: string;
  user_id?: string;
}

export interface ModelsUser {
  addresses?: Array<ModelsShippingAddress>;
  auth_providers?: Array<ModelsAuthProvider>;
  avatar_url?: string;
  created_at?: string;
  email?: string;
  email_verified?: boolean;
  id?: string;
  is_active?: boolean;
  role?: ModelsUserRole;
  updated_at?: string;
  username?: string;
}

export type ModelsUserRole = "viewer" | "streamer" | "agency" | "admin";

export interface ServicesAddressInput {
  address_line1: string;
  address_line2?: string;
  city: string;
  country?: string;
  district?: string;
  is_default?: boolean;
  phone?: string;
  postal_code?: string;
  recipient_name: string;
}

export interface ServicesClaimInput {
  address_line1: string;
  address_line2?: string;
  city: string;
  country?: string;
  phone?: string;
  postal_code?: string;
  recipient_name: string;
}

export interface ServicesLinkWalletInput {
  address: string;
  nonce: string;
  signature: string;
}

export interface ServicesLoginInput {
  email: string;
  password: string;
}

export interface ServicesRegisterInput {
  email: string;
  password: string;
  username: string;
}

export interface ServicesUpdateProfileInput {
  avatar_url?: string;
  username?: string;
}

export interface ServicesWeb3VerifyInput {
  address: string;
  nonce: string;
  signature: string;
}

export interface ApiOperations {
  "POST /auth/forgot-password": {
    requestBody: {
      email?: string;
    };
    response: HandlersResponse & {
      data?: HandlersMessageResponse;
    };
  };
  "GET /auth/google": {
    response: unknown;
  };
  "GET /auth/google/callback": {
    queryParams: {
      code: string;
      state: string;
    };
    response: HandlersResponse & {
      data?: HandlersAuthResponse;
    };
  };
  "POST /auth/login": {
    requestBody: ServicesLoginInput;
    response: HandlersResponse & {
      data?: HandlersAuthResponse;
    };
  };
  "POST /auth/logout": {
    requestBody?: {
      refresh_token?: string;
    };
    response: HandlersResponse & {
      data?: HandlersMessageResponse;
    };
  };
  "DELETE /auth/providers/{provider}": {
    pathParams: {
      provider: "twitch" | "google" | "web3" | "email";
    };
    response: HandlersResponse & {
      data?: HandlersMessageResponse;
    };
  };
  "POST /auth/refresh": {
    requestBody?: {
      refresh_token?: string;
    };
    response: HandlersResponse & {
      data?: HandlersTokensResponse;
    };
  };
  "POST /auth/register": {
    requestBody: ServicesRegisterInput;
    response: HandlersResponse & {
      data?: HandlersAuthResponse;
    };
  };
  "POST /auth/reset-password": {
    requestBody: {
      new_password?: string;
      token?: string;
    };
    response: HandlersResponse & {
      data?: HandlersMessageResponse;
    };
  };
  "GET /auth/twitch": {
    response: unknown;
  };
  "GET /auth/twitch/callback": {
    queryParams: {
      code: string;
      state: string;
    };
    response: HandlersResponse & {
      data?: HandlersAuthResponse;
    };
  };
  "POST /auth/verify-email/confirm": {
    requestBody: {
      token?: string;
    };
    response: HandlersResponse & {
      data?: HandlersMessageResponse;
    };
  };
  "POST /auth/verify-email/send": {
    response: HandlersResponse & {
      data?: HandlersMessageResponse;
    };
  };
  "POST /auth/web3/nonce": {
    requestBody: {
      address?: string;
    };
    response: HandlersResponse & {
      data?: HandlersNonceResponse;
    };
  };
  "POST /auth/web3/verify": {
    requestBody: ServicesWeb3VerifyInput;
    response: HandlersResponse & {
      data?: HandlersAuthResponse;
    };
  };
  "GET /claim/{token}": {
    pathParams: {
      token: string;
    };
    response: HandlersResponse;
  };
  "POST /claim/{token}": {
    requestBody: ServicesClaimInput;
    pathParams: {
      token: string;
    };
    response: HandlersResponse;
  };
  "GET /dashboard/raffles": {
    response: HandlersResponse;
  };
  "POST /dashboard/raffles": {
    requestBody: {
      title?: string;
    };
    response: HandlersResponse;
  };
  "GET /dashboard/raffles/{id}": {
    pathParams: {
      id: string;
    };
    response: HandlersResponse;
  };
  "POST /dashboard/raffles/{id}/activate": {
    pathParams: {
      id: string;
    };
    response: HandlersResponse;
  };
  "POST /dashboard/raffles/{id}/complete": {
    pathParams: {
      id: string;
    };
    response: HandlersResponse;
  };
  "PATCH /dashboard/raffles/{id}/discord-webhook": {
    requestBody: {
      discord_webhook_url?: string;
    };
    pathParams: {
      id: string;
    };
    response: HandlersResponse;
  };
  "GET /dashboard/raffles/{id}/draws": {
    pathParams: {
      id: string;
    };
    response: HandlersResponse;
  };
  "POST /dashboard/raffles/{id}/draws": {
    pathParams: {
      id: string;
    };
    response: HandlersResponse;
  };
  "POST /dashboard/raffles/{id}/entries/import-csv": {
    pathParams: {
      id: string;
    };
    response: HandlersResponse;
  };
  "POST /dashboard/raffles/{id}/snapshot": {
    requestBody: {
      source?: string;
    };
    pathParams: {
      id: string;
    };
    response: HandlersResponse;
  };
  "POST /extension/auth/login": {
    requestBody: {
      extension_jwt?: string;
    };
    response: HandlersResponse & {
      data?: HandlersAuthResponse;
    };
  };
  "POST /extension/bits/complete": {
    requestBody: {
      extension_jwt?: string;
      sku?: string;
      transaction_receipt?: string;
    };
    response: HandlersResponse & {
      data?: HandlersAuthResponse;
    };
  };
  "GET /extension/raffles/{id}/result": {
    pathParams: {
      id: string;
    };
    response: HandlersResponse;
  };
  "POST /extension/t-point/complete": {
    requestBody: {
      extension_jwt?: string;
      sku?: string;
      transaction_receipt?: string;
    };
    response: HandlersResponse & {
      data?: HandlersAuthResponse;
    };
  };
  "POST /spend/redeem": {
    requestBody: HandlersRedeemRequest;
    response: HandlersResponse & {
      data?: HandlersRedeemResponse;
    };
  };
  "GET /users/me": {
    response: HandlersResponse & {
      data?: HandlersUserResponse;
    };
  };
  "PUT /users/me": {
    requestBody: ServicesUpdateProfileInput;
    response: HandlersResponse & {
      data?: HandlersUserResponse;
    };
  };
  "GET /users/me/addresses": {
    response: HandlersResponse & {
      data?: HandlersAddressesResponse;
    };
  };
  "POST /users/me/addresses": {
    requestBody: ServicesAddressInput;
    response: HandlersResponse & {
      data?: HandlersAddressResponse;
    };
  };
  "PUT /users/me/addresses/{id}": {
    requestBody: ServicesAddressInput;
    pathParams: {
      id: string;
    };
    response: HandlersResponse & {
      data?: HandlersAddressResponse;
    };
  };
  "DELETE /users/me/addresses/{id}": {
    pathParams: {
      id: string;
    };
    response: HandlersResponse & {
      data?: HandlersMessageResponse;
    };
  };
  "PUT /users/me/addresses/{id}/default": {
    pathParams: {
      id: string;
    };
    response: HandlersResponse & {
      data?: HandlersAddressResponse;
    };
  };
  "GET /users/me/points": {
    queryParams: {
      channel_id: string;
    };
    response: HandlersResponse & {
      data?: HandlersPointsBalanceResponse;
    };
  };
  "POST /users/me/points/claim": {
    requestBody?: HandlersClaimRequest;
    response: HandlersResponse & {
      data?: HandlersTachiBalanceResponse;
    };
  };
  "GET /users/me/points/history": {
    queryParams: {
      channel_id: string;
    };
    response: HandlersResponse & {
      data?: HandlersPointsHistoryResponse;
    };
  };
  "GET /users/me/providers": {
    response: HandlersResponse & {
      data?: HandlersProvidersResponse;
    };
  };
  "GET /users/me/tachi/balance": {
    response: HandlersResponse & {
      data?: HandlersTachiBalanceResponse;
    };
  };
  "POST /users/me/wallet": {
    requestBody: ServicesLinkWalletInput;
    response: HandlersResponse & {
      data?: HandlersWalletResponse;
    };
  };
}
