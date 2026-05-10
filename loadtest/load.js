import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, pick, SEARCH_TERMS, CATEGORIES, LOCATIONS } from './common.js';

export { setup } from './common.js';

export const options = {
  scenarios: {
    browse: {
      executor: 'ramping-vus',
      exec:     'browseScenario',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 50  },
        { duration: '1m',  target: 100 },
        { duration: '5m',  target: 100 },
        { duration: '30s', target: 0   },
      ],
    },
    writes: {
      executor: 'ramping-vus',
      exec:     'writeScenario',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 10 },
        { duration: '1m',  target: 30 },
        { duration: '5m',  target: 30 },
        { duration: '30s', target: 0  },
      ],
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<500'],
    http_req_failed:   ['rate<0.05'],
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

  sleep(1 + Math.random() * 2);
}

export function writeScenario(data) {
  const res = http.post(
    `${BASE_URL}/api/v1/listings`,
    JSON.stringify({
      user_id:     pick(data.userIds),
      title:       `Load test ${Date.now()}`,
      description: 'load test listing',
      price:       Math.floor(Math.random() * 100000) + 1000,
      category:    pick(CATEGORIES),
      location:    pick(LOCATIONS),
    }),
    { headers: { 'Content-Type': 'application/json' } }
  );
  check(res, { 'create 201': (r) => r.status === 201 });
  sleep(2 + Math.random() * 3);
}
