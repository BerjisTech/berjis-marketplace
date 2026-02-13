type MarketplaceWindow = Window &
  typeof globalThis & {
    __MARKETPLACE_API__?: string;
    __CORE_API__?: string;
  };

const w: MarketplaceWindow | undefined = typeof window !== 'undefined' ? (window as MarketplaceWindow) : undefined;
const loc = typeof location !== 'undefined' ? location : null;
const defaultMarketplace = typeof w?.__MARKETPLACE_API__ === 'string'
  ? w?.__MARKETPLACE_API__ as string
  : loc
    ? `${loc.protocol}//marketplace-api.berjis.tech`
    : 'https://marketplace-api.berjis.tech';
const defaultCore = typeof w?.__CORE_API__ === 'string'
  ? w?.__CORE_API__ as string
  : (loc && (loc.hostname === 'localhost' || loc.hostname === '127.0.0.1'))
    ? 'http://localhost:8080'
    : 'https://api.berjis.tech';

export const environment = {
  production: false,
  apiBase: defaultMarketplace,
  marketplaceApi: defaultMarketplace,
  coreApi: defaultCore
};
