import { defineCollection, z } from 'astro:content';

const dateValue = z.union([z.date(), z.string()]).transform((value) => {
  if (value instanceof Date) return value;
  return new Date(value);
}).refine((value) => !Number.isNaN(value.getTime()), {
  message: 'Post date must be a valid date or ISO date string'
});

const booleanValue = z.union([z.boolean(), z.string()]).optional().default(false).transform((value) => {
  return value === true || value === 'true';
});

const posts = defineCollection({
  type: 'content',
  schema: z.object({
    title: z.string(),
    description: z.string().optional().default(''),
    author: z.string().optional().default('honemik'),
    date: dateValue,
    tags: z.array(z.string()).optional().default([]),
    categories: z.array(z.string()).optional().default([]),
    series: z.array(z.string()).optional().default([]),
    draft: booleanValue,
    ShowToc: z.boolean().optional().default(false),
    ShowBreadCrumbs: z.boolean().optional().default(true)
  }).passthrough()
});

export const collections = { posts };
