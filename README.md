# My Personal Blog

This is my personal blog, built with [Hugo](https://gohugo.io/) and the [re-Terminal](https://github.com/mirus-ua/hugo-theme-re-terminal) theme. The blog is automatically deployed to [GitHub Pages](https://pages.github.com/) using GitHub Actions.

## Technology Stack

*   **Static Site Generator:** [Hugo](https://gohugo.io/)
*   **Theme:** [re-Terminal](https://github.com/mirus-ua/hugo-theme-re-terminal)
*   **Hosting:** [GitHub Pages](https://pages.github.com/)
*   **CI/CD:** [GitHub Actions](https://github.com/features/actions)

## Local Development

To run the blog locally, you need to have Hugo installed. You can find the installation instructions [here](https://gohugo.io/getting-started/installing/).

Once you have Hugo installed, you can run the following command to start the local server:

```bash
hugo server
```

The blog will be available at `http://localhost:1313/`.

## Creating a New Post

To create a new post, you can use the following command:

```bash
hugo new content posts/your-new-post-title.md
```

This will create a new Markdown file in the `content/posts` directory. You can then open the file and start writing your post.

## Deployment

This blog is automatically deployed to GitHub Pages whenever a new commit is pushed to the `main` branch. The deployment process is handled by a GitHub Actions workflow defined in `.github/workflows/deploy.yml`.
