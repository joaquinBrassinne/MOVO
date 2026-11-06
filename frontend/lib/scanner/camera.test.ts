import assert from "node:assert/strict";
import { test } from "node:test";

import { CAMERA_PROBLEM, cameraProblemOf } from "./camera.ts";

const named = (name: string) => Object.assign(new Error("x"), { name });

test("a denied permission is explained and points to the manual entry", () => {
  for (const name of ["NotAllowedError", "SecurityError"]) {
    const problem = cameraProblemOf(named(name));

    assert.equal(problem.kind, CAMERA_PROBLEM.DENIED, name);
    assert.match(problem.body, /ingresar el código a mano/);
  }
});

test("a device without a camera, or a browser without camera support, is explained", () => {
  assert.equal(cameraProblemOf(named("NotFoundError")).kind, CAMERA_PROBLEM.UNAVAILABLE);
  assert.equal(cameraProblemOf(named("OverconstrainedError")).kind, CAMERA_PROBLEM.UNAVAILABLE);
  assert.equal(cameraProblemOf(undefined).kind, CAMERA_PROBLEM.UNAVAILABLE);
});

test("any other failure never leaks the browser's text", () => {
  const problem = cameraProblemOf(
    Object.assign(new Error("secret detail"), { name: "AbortError" }),
  );

  assert.equal(problem.kind, CAMERA_PROBLEM.FAILED);
  assert.doesNotMatch(`${problem.title} ${problem.body}`, /secret|AbortError/);
  assert.match(problem.body, /ingresar el código a mano/);
});
