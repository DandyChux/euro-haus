import apiClient from "$lib/api";
import type { IssueSubmission } from "$lib/schemas/submission";
import type { PageLoad } from "../$types";

export const load: PageLoad = async () => {
	const response = await apiClient.get<{
		submissions?: IssueSubmission[];
	}>("/admin/submissions/issues");

	return {
		submissions: response.submissions ?? [],
	};
};
