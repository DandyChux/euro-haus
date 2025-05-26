import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Form, FormItem, FormLabel, FormControl, FormMessage, FormField } from "~/components/ui/form";
import { Button } from "~/components/ui/button";
import { Input } from "~/components/ui/input";
import { Textarea } from "~/components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "~/components/ui/select";

const phoneRegex = new RegExp(
	/^([+]?[\s0-9]+)?(\d{3}|[(]?[0-9]+[)])?([-]?[\s]?[0-9])+$/
);

const subjects = ['General Inquiry', 'Event Information', 'Partnership/Collaboration', 'Merchandise Question', 'Media/Press Inquiry', 'Feedback', 'Other'] as const

const subjectsEnum = z.enum(subjects)

const contactFormSchema = z.object({
	name: z.string().min(2).max(100),
	email: z.string().email('Invalid email address'),
	phone: z.string().regex(phoneRegex, 'Invalid phone number'),
	subject: subjectsEnum,
	message: z.string().min(10, 'Message must be at least 10 characters long').max(500, 'Message cannot exceed 500 characters')
})

type ContactForm = z.infer<typeof contactFormSchema>

export function ContactForm() {
	const form = useForm<ContactForm>({
		resolver: zodResolver(contactFormSchema),
		defaultValues: {
			name: "",
			email: "",
			phone: "",
			subject: subjects[0],
			message: ""
		}
	})

	function onSubmit(data: ContactForm) {
		// Send email with this data
		console.log(data);
	}

	return (
		<Form {...form}>
			<form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
				<div className='inline-flex space-x-4 w-full'>
					<FormField
						control={form.control}
						name='name'
						render={({ field }) => (
							<FormItem className='flex-1'>
								<FormLabel>Name</FormLabel>
								<FormControl>
									<Input placeholder='Your name' {...field} />
								</FormControl>
								<FormMessage />
							</FormItem>
						)}
					/>
					<FormField
						control={form.control}
						name='email'
						render={({ field }) => (
							<FormItem className='flex-1'>
								<FormLabel>Email</FormLabel>
								<FormControl>
									<Input placeholder='Your email address' {...field} />
								</FormControl>
								<FormMessage />
							</FormItem>
						)}
					/>
				</div>
				<FormField
					control={form.control}
					name='subject'
					render={({ field }) => (
						<FormItem>
							<FormLabel>Subject</FormLabel>

							<Select onValueChange={field.onChange} defaultValue={field.value}>
								<FormControl>
									<SelectTrigger>
										<SelectValue placeholder='Select a subject' />
									</SelectTrigger>
								</FormControl>
								<SelectContent>
									{subjects.map(subject => (
										<SelectItem key={subject} value={subject}>
											{subject}
										</SelectItem>
									))}
								</SelectContent>
							</Select>
							<FormMessage />
						</FormItem>
					)}
				/>
				<FormField
					control={form.control}
					name='phone'
					render={({ field }) => (
						<FormItem>
							<FormLabel>Phone</FormLabel>
							<FormControl>
								<Input
									placeholder='(123)456-7890'
									inputMode='tel'
									pattern={phoneRegex.source}
									// type="number"
									{...field}
								/>
							</FormControl>
							<FormMessage />
						</FormItem>
					)}
				/>
				<FormField
					control={form.control}
					name='message'
					render={({ field }) => (
						<FormItem>
							<FormLabel>Message</FormLabel>
							<FormControl>
								<Textarea placeholder='Your message' {...field} />
							</FormControl>
							<FormMessage />
						</FormItem>
					)}
				/>

				<Button type='submit' disabled={!form.formState.isValid}>
					{form.formState.isSubmitting ? 'Sending...' : 'Send Message'}
				</Button>
			</form>
		</Form>
	)
}
