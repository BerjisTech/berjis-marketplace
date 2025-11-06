export const environment = {
  production: false,
  apiBase: (typeof location !== 'undefined') ? `${location.protocol}//marketplace-api.berjis.tech` : 'https://marketplace-api.berjis.tech'
};

