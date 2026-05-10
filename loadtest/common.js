import http from 'k6/http';

export const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export const SEARCH_TERMS = [
  'iPhone', 'Ноутбук', 'Диван', 'Велосипед', 'Куртка',
  'Телевизор', 'Холодильник', 'Стол', 'Кресло', 'Планшет',
  'Новый', 'Срочно', 'Москва', 'Электроника', 'Авто',
];

export const CATEGORIES = ['Электроника', 'Авто', 'Недвижимость', 'Одежда', 'Спорт', 'Мебель'];
export const LOCATIONS  = ['Москва', 'Санкт-Петербург', 'Казань', 'Екатеринбург', 'Новосибирск'];

export function setup() {
  const listingIds = [];
  const userIdsSet = {};

  for (const term of SEARCH_TERMS) {
    const res = http.get(`${BASE_URL}/api/v1/listings/search?q=${encodeURIComponent(term)}&limit=100`);
    if (res.status !== 200) continue;

    const body = JSON.parse(res.body);
    for (const item of (body.items || [])) {
      if (item.id)      listingIds.push(item.id);
      if (item.user_id) userIdsSet[item.user_id] = true;
    }
  }

  const userIds = Object.keys(userIdsSet);
  if (listingIds.length === 0) {
    throw new Error('setup: no listings found — seed the database first');
  }

  return {
    listingIds: listingIds.slice(0, 500),
    userIds:    userIds.slice(0, 100),
  };
}

export function pick(arr) {
  return arr[Math.floor(Math.random() * arr.length)];
}
