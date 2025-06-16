export default {
	name: 'event',
	title: 'Events',
	type: 'document',
	fields: [
		{
			name: 'title',
			title: 'Event Title',
			type: 'string',
			validation: (Rule: any) => Rule.required(),
		},
		{
			name: 'slug',
			title: 'Slug',
			type: 'slug',
			description: 'URL-friendly version of the title',
			options: {
				source: 'title',
				maxLength: 96,
			},
			validation: (Rule: any) => Rule.required(),
		},
		{
			name: 'date',
			title: 'Event Date',
			type: 'datetime',
			validation: (Rule: any) => Rule.required(),
		},
		{
			name: 'description',
			title: 'Description',
			type: 'text',
			rows: 4,
			validation: (Rule: any) => Rule.required(),
		},
		{
			name: 'longDescription',
			title: 'Detailed Description',
			type: 'array',
			of: [{ type: 'block' }],
			description: 'Rich text description for the event detail page',
		},
		{
			name: 'location',
			title: 'Location',
			type: 'object',
			fields: [
				{
					name: 'name',
					title: 'Venue Name',
					type: 'string',
					validation: (Rule: any) => Rule.required(),
				},
				{
					name: 'address',
					title: 'Address',
					type: 'string',
				},
				{
					name: 'city',
					title: 'City',
					type: 'string',
				},
				{
					name: 'state',
					title: 'State',
					type: 'string',
				},
				{
					name: 'coordinates',
					title: 'Coordinates',
					type: 'geopoint',
				},
			],
		},
		{
			name: 'price',
			title: 'Starting Price',
			type: 'number',
			validation: (Rule: any) => Rule.required().positive(),
		},
		{
			name: 'image',
			title: 'Main Image',
			type: 'image',
			options: {
				hotspot: true,
			},
			fields: [
				{
					name: 'alt',
					title: 'Alt Text',
					type: 'string',
				},
			],
			validation: (Rule: any) => Rule.required(),
		},
		{
			name: 'gallery',
			title: 'Image Gallery',
			type: 'array',
			of: [
				{
					type: 'image',
					options: {
						hotspot: true,
					},
					fields: [
						{
							name: 'alt',
							title: 'Alt Text',
							type: 'string',
						},
					],
				},
			],
		},
		{
			name: 'capacity',
			title: 'Total Capacity',
			type: 'number',
			validation: (Rule: any) => Rule.positive(),
		},
		{
			name: 'availableSpots',
			title: 'Available Spots',
			type: 'number',
			validation: (Rule: any) => Rule.positive(),
		},
		{
			name: 'organizer',
			title: 'Organizer',
			type: 'string',
			initialValue: 'Euro Haus Events Team',
		},
		{
			name: 'tags',
			title: 'Tags',
			type: 'array',
			of: [{ type: 'string' }],
			options: {
				layout: 'tags',
			},
		},
		{
			name: 'status',
			title: 'Event Status',
			type: 'string',
			options: {
				list: [
					{ title: 'Upcoming', value: 'upcoming' },
					{ title: 'Ongoing', value: 'ongoing' },
					{ title: 'Completed', value: 'completed' },
					{ title: 'Cancelled', value: 'cancelled' },
					{ title: 'Sold Out', value: 'soldout' },
				],
				layout: 'dropdown',
			},
			initialValue: 'upcoming',
		},
		{
			name: 'agenda',
			title: 'Event Schedule',
			type: 'array',
			of: [
				{
					type: 'object',
					fields: [
						{
							name: 'time',
							title: 'Time',
							type: 'string',
						},
						{
							name: 'activity',
							title: 'Activity',
							type: 'string',
						},
					],
				},
			],
		},
		{
			name: 'includes',
			title: "What's Included",
			type: 'array',
			of: [{ type: 'string' }],
		},
		{
			name: 'featured',
			title: 'Featured Event',
			type: 'boolean',
			description: 'Show this event on the homepage',
			initialValue: false,
		},
		{
			name: 'registrationLink',
			title: 'Registration Link',
			type: 'url',
			description: 'External registration link (if applicable)',
		},
	],
	preview: {
		select: {
			title: 'title',
			date: 'date',
			media: 'image',
			status: 'status',
		},
		prepare(selection: any) {
			const { title, date, media, status } = selection;
			return {
				title,
				subtitle: `${new Date(date).toLocaleDateString()} - ${status}`,
				media,
			};
		},
	},
};
