type MarketplaceWindow = Window &
  typeof globalThis & {
    __MARKETPLACE_API__?: string;
    __CORE_API__?: string;
  };

const w: MarketplaceWindow | undefined = typeof window !== 'undefined' ? (window as MarketplaceWindow) : undefined;
const defaultMarketplace = typeof w?.__MARKETPLACE_API__ === 'string'
  ? (w?.__MARKETPLACE_API__ as string)
  : 'https://marketplace-api.berjis.tech';
const defaultCore = typeof w?.__CORE_API__ === 'string'
  ? (w?.__CORE_API__ as string)
  : 'https://api.berjis.tech';

export const environment = {
  production: true,
  apiBase: defaultMarketplace,
  marketplaceApi: defaultMarketplace,
  coreApi: defaultCore
};
