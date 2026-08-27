// k6 load test — distributed job processing platform
//
// install:  https://k6.io/docs/getting-started/installation/
// run:      k6 run loadtest.js
// options:  k6 run loadtest.js --vus 50 --duration 2m
//
// this script:
//   1. registers a test user (once per vu)
//   2. logs in and gets a jwt
//   3. submits jobs in a loop
//   4. polls for job completion
//   5. checks response times and error rates

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// ── configuration ────────────────────────────────────────────────────────────

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export const options = {
  scenarios: {
    // ramp up to 30 vus over 1 minute, hold for 3 minutes, ramp down.
    load_test: {
      executor: 'ramping-vus',
      startVUs: 1,
      stages: [
        { duration: '30s', target: 10 },
        { duration: '1m',  target: 30 },
        { duration: '2m',  target: 30 },
        { duration: '30s', target: 0 },
      ],
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'], // 95th < 500ms, 99th < 1s
    http_req_failed:   ['rate<0.05'],                 // Error rate < 5%
    job_creation_time: ['p(95)<300'],
  },
};

// ── custom metrics ───────────────────────────────────────────────────────────

const jobCreationTime = new Trend('job_creation_time');
const jobErrors = new Rate('job_errors');

// ── setup (once per test run) ────────────────────────────────────────────────

export function setup() {
  // health check
  const healthRes = http.get(`${BASE_URL}/health/live`);
  check(healthRes, { 'API is alive': (r) => r.status === 200 });

  return {};
}

// ── per-vu init ──────────────────────────────────────────────────────────────

let token = '';
let userId = '';

function ensureAuth() {
  if (token) return;

  const vuEmail = `loadtest-vu${__VU}-${Date.now()}@test.com`;
  const password = 'LoadTest123!';

  // register
  const regRes = http.post(
    `${BASE_URL}/api/v1/auth/register`,
    JSON.stringify({ email: vuEmail, password }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  if (regRes.status === 201) {
    userId = regRes.json('id');
  }

  // login
  const loginRes = http.post(
    `${BASE_URL}/api/v1/auth/login`,
    JSON.stringify({ email: vuEmail, password }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  check(loginRes, { 'login successful': (r) => r.status === 200 });

  if (loginRes.status === 200) {
    token = loginRes.json('access_token');
  }
}

function authHeaders() {
  return {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`,
  };
}

// ── main test logic ──────────────────────────────────────────────────────────

export default function () {
  ensureAuth();

  if (!token) {
    jobErrors.add(1);
    sleep(1);
    return;
  }

  group('Job Lifecycle', () => {
    // 1. create a job
    const jobTypes = ['report_generation', 'data_processing'];
    const jobType = jobTypes[Math.floor(Math.random() * jobTypes.length)];

    const createStart = Date.now();
    const createRes = http.post(
      `${BASE_URL}/api/v1/jobs`,
      JSON.stringify({
        type: jobType,
        priority: Math.floor(Math.random() * 10) + 1,
        payload: { test: true, vu: __VU, iter: __ITER },
        max_retries: 3,
      }),
      {
        headers: {
          ...authHeaders(),
          'Idempotency-Key': `load-${__VU}-${__ITER}-${Date.now()}`,
        },
      }
    );

    jobCreationTime.add(Date.now() - createStart);

    const created = check(createRes, {
      'job created': (r) => r.status === 201 || r.status === 200,
    });

    if (!created) {
      jobErrors.add(1);
      return;
    }

    const jobId = createRes.json('id');

    // 2. get job details
    const getRes = http.get(`${BASE_URL}/api/v1/jobs/${jobId}`, {
      headers: authHeaders(),
    });

    check(getRes, { 'job fetched': (r) => r.status === 200 });

    // 3. list jobs
    const listRes = http.get(`${BASE_URL}/api/v1/jobs?page=1&limit=10`, {
      headers: authHeaders(),
    });

    check(listRes, { 'jobs listed': (r) => r.status === 200 });
  });

  group('Health Checks', () => {
    const liveRes = http.get(`${BASE_URL}/health/live`);
    check(liveRes, { 'liveness ok': (r) => r.status === 200 });

    const readyRes = http.get(`${BASE_URL}/health/ready`);
    check(readyRes, { 'readiness ok': (r) => r.status === 200 });
  });

  sleep(Math.random() * 2 + 0.5); // 0.5–2.5s between iterations
}

// ── teardown ─────────────────────────────────────────────────────────────────

export function teardown(data) {
  console.log('Load test completed.');
}
