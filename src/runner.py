class Runner:
    def __init__(self, database, job):
        self.database = database
        self.last_job_at = self.database.get_last_job_at(job)

    def store_report(self, job, elapsed_time):
        self.database.create_report({"job": job, "elapsed_time": elapsed_time})
