import http from 'k6/http';
import { check, sleep } from 'k6';
import { setup as commonSetup, BASE_URL, pick, SEARCH_TERMS } from './common.js';

export { setup } from './common.js';

export const options = {
  vus: 5,
  duration: '30s',
  thresholds: {
    http_req_duration: ['p(95)<500'],
    http_req_failed:   ['rate<0.01'],
  },
};

export default function (data) {
  const listingId = pick(data.listingIds);
  const userId    = pick(data.userIds);
  const term      = encodeURIComponent(pick(SEARCH_TERMS));

  check(http.get(`${BASE_URL}/api/v1/listings/${listingId}`), {
    'get card 200':   (r) => r.status === 200,
    'has title':      (r) => JSON.parse(r.body).title !== undefined,
  });

  check(http.get(`${BASE_URL}/api/v1/listings/search?q=${term}&limit=10`), {
    'search 200':     (r) => r.status === 200,
    'has total':      (r) => JSON.parse(r.body).total !== undefined,
  });

  check(http.get(`${BASE_URL}/api/v1/users/${userId}/listings?per_page=10`), {
    'user listings 200': (r) => r.status === 200,
  });

  sleep(1);
}
