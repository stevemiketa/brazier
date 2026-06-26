import { setApiKey, getApiKey, clearApiKey, isAuthenticated } from "./auth";

beforeEach(() => {
  clearApiKey();
});

test("isAuthenticated returns false when no key set", () => {
  expect(isAuthenticated()).toBe(false);
});

test("setApiKey / getApiKey round-trips", () => {
  setApiKey("bz_abc123");
  expect(getApiKey()).toBe("bz_abc123");
  expect(isAuthenticated()).toBe(true);
});

test("clearApiKey removes the key", () => {
  setApiKey("bz_abc123");
  clearApiKey();
  expect(getApiKey()).toBeNull();
  expect(isAuthenticated()).toBe(false);
});
