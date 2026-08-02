import type { PageLoad } from "./$types";
import { getCatalogProducts } from "$lib/services/stripe";

export const load: PageLoad = async ({ fetch }) => {
  return {
    products: await getCatalogProducts(fetch),
    title: "Catalog",
  };
};
