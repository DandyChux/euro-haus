import { error } from "@sveltejs/kit";
import type { PageLoad } from "./$types";
import { getCheckoutSession } from "$lib/services/stripe";

export const load: PageLoad = async ({ fetch, url }) => {
  const sessionId = url.searchParams.get("session_id");

  if (!sessionId) {
    error(400, "Missing session_id");
  }

  return {
    order: await getCheckoutSession(fetch, sessionId),
    title: "Order successful",
  };
};
