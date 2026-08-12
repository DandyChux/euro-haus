import apiClient from "$lib/api";
import type { IssueSubmission } from "$lib/schemas/submission";
import type { PageLoad } from "./$types";

interface IssuesResponse {
	submissions?: IssueSubmission[];
}

export const load: PageLoad = async () => {
	const response = await apiClient.get<IssuesResponse>(
		"/admin/submissions/issues",
	);

	return {
		issues: response.submissions ?? [],
	};
};
