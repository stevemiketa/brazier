import React from "react";
import { render, screen } from "@testing-library/react";
import { StatusBadge } from "./StatusBadge";

test("renders success state", () => {
  render(<StatusBadge state="success" />);
  expect(screen.getByText("success")).toBeTruthy();
});

test("renders failed state", () => {
  render(<StatusBadge state="failed" />);
  expect(screen.getByText("failed")).toBeTruthy();
});

test("renders pending state", () => {
  render(<StatusBadge state="pending" />);
  expect(screen.getByText("pending")).toBeTruthy();
});
