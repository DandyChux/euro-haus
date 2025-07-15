import { useState } from 'react';
import { Button } from './button';
import { Textarea } from './textarea';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from './card';
import { Label } from './label';
import { Loader2 } from 'lucide-react';
import { Switch } from './switch';
import { toast } from 'sonner';
import { apiClient } from '~/lib/api';

interface TextContentEditorProps {
	placementId: string;
	currentText: string;
	defaultText: string;
	placementName: string;
	isHtml?: boolean;
	onSave?: () => void;
	onCancel: () => void;
}

export function TextContentEditor({
	placementId,
	currentText,
	defaultText,
	placementName,
	isHtml = false,
	onSave,
	onCancel
}: TextContentEditorProps) {
	const [text, setText] = useState(currentText || defaultText);
	const [isSubmitting, setIsSubmitting] = useState(false);
	const [showPreview, setShowPreview] = useState(false);

	const handleSave = async () => {
		setIsSubmitting(true);
		try {
			await apiClient.put(`/content-placements/${placementId}`, {
				textContent: text
			});
			toast.success('Text updated', {
				description: 'The text content has been updated successfully.'
			})
			if (onSave) onSave();
		} catch (error) {
			console.error('Error updating text content:', error);
			toast.error('Update failed', {
				description: "There was an error updating the text content."
			})
		} finally {
			setIsSubmitting(false);
		}
	};

	return (
		<Card className="w-full">
			<CardHeader>
				<CardTitle>Edit Text Content</CardTitle>
				<CardDescription>{placementName}</CardDescription>
			</CardHeader>
			<CardContent>
				<div className="space-y-4">
					{isHtml && (
						<div className="flex items-center space-x-2">
							<Switch
								id="preview-mode"
								checked={showPreview}
								onCheckedChange={setShowPreview}
							/>
							<Label htmlFor="preview-mode">Preview HTML</Label>
						</div>
					)}
					{showPreview && isHtml ? (
						<div className="border rounded-md p-4 min-h-[200px]">
							<div dangerouslySetInnerHTML={{ __html: text }} />
						</div>
					) : (
						<Textarea
							value={text}
							onChange={(e) => setText(e.target.value)}
							placeholder="Enter text content..."
							className="min-h-[200px]"
						/>
					)}

					{isHtml && (
						<p className="text-sm text-muted-foreground">
							This content supports HTML tags. Use with caution.
						</p>
					)}

					<div className="flex justify-between items-center pt-2">
						<Button
							variant="ghost"
							onClick={() => setText(defaultText)}
							type="button"
						>
							Reset to Default
						</Button>
					</div>
				</div>
			</CardContent>
			<CardFooter className="flex justify-between">
				<Button variant="outline" onClick={onCancel}>
					Cancel
				</Button>
				<Button onClick={handleSave} disabled={isSubmitting}>
					{isSubmitting ? (
						<>
							<Loader2 className="mr-2 h-4 w-4 animate-spin" />
							Saving...
						</>
					) : (
						'Save Changes'
					)}
				</Button>
			</CardFooter>
		</Card>
	);
}
