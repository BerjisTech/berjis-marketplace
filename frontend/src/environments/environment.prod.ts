const w: any = typeof window !== 'undefined' ? window : {};
const defaultMarketplace = typeof w.__MARKETPLACE_API__ === 'string'
  ? w.__MARKETPLACE_API__
  : 'https://marketplace-api.berjis.tech';
const defaultCore = typeof w.__CORE_API__ === 'string'
  ? w.__CORE_API__
  : 'https://api.berjis.tech';

export const environment = {
  production: true,
  apiBase: defaultMarketplace,
  marketplaceApi: defaultMarketplace,
  coreApi: defaultCore
};
