/** Response of POST /admin/password-resets: the token is shown only once. */
export interface IssuedReset {
  user_id: number;
  email: string;
  token: string;
  expires_at: string;
}
