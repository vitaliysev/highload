import http from 'k6/http';
import { check } from 'k6';
import { BASE_URL, pick, SEARCH_TERMS, CATEGORIES, LOCATIONS } from './common.js';

export { setup } from './common.js';

export const options = {
  scenarios: {
    browse: {
      executor: 'ramping-arrival-rate',
      exec:     'browseScenario',
      startRate: 0,
      timeUnit: '1s',
      preAllocatedVUs: 160,
      maxVUs: 320,
      stages: [
        { duration: '30s', target: 66  },
        { duration: '30s', target: 132 },
        { duration: '30s', target: 176 },
        { duration: '3m',  target: 176 },
        { duration: '30s', target: 0   },
      ],
    },
    writes: {
      executor: 'ramping-arrival-rate',
      exec:     'writeScenario',
      startRate: 0,
      timeUnit: '1s',
      preAllocatedVUs: 44,
      maxVUs: 88,
      stages: [
        { duration: '30s', target: 16 },
        { duration: '30s', target: 33 },
        { duration: '30s', target: 44 },
        { duration: '3m',  target: 44 },
        { duration: '30s', target: 0  },
      ],
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<2000'],
    http_req_failed:   ['rate<0.20'],
  },
};

export function browseScenario(data) {
  const roll = Math.random();
  let res;

  if (roll < 0.50) {
    res = http.get(`${BASE_URL}/api/v1/listings/${pick(data.listingIds)}`);
    check(res, { 'get card 200': (r) => r.status === 200 });
  } else if (roll < 0.85) {
    const q = encodeURIComponent(pick(SEARCH_TERMS));
    res = http.get(`${BASE_URL}/api/v1/listings/search?q=${q}&limit=20`);
    check(res, { 'search 200': (r) => r.status === 200 });
  } else {
    res = http.get(`${BASE_URL}/api/v1/users/${pick(data.userIds)}/listings?per_page=20`);
    check(res, { 'user listings 200': (r) => r.status === 200 });
  }
}

export function writeScenario(data) {
  const res = http.post(
    `${BASE_URL}/api/v1/listings`,
    JSON.stringify({
      user_id:     pick(data.userIds),
      title:       `Stress test ${Date.now()}`,
      description: 'stress',
      price:       Math.floor(Math.random() * 50000) + 500,
      category:    pick(CATEGORIES),
      location:    pick(LOCATIONS),
    }),
    { headers: { 'Content-Type': 'application/json' } }
  );
  check(res, { 'create 201': (r) => r.status === 201 });
}
