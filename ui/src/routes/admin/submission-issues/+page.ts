import apiClient from "$lib/api";
import type { IssueSubmission } from "$lib/schemas/submission";
import type { PageLoad } from "./$types";

interface IssuesResponse {
	submissions: IssueSubmission[];
	total: number;
}

export const load: PageLoad = async ({ fetch }) => {
	const response = await apiClient.get<IssuesResponse>(
		"/admin/submissions/issues",
		undefined,
		fetch,
	);

	return {
		issues: response.submissions ?? [],
	};
};
